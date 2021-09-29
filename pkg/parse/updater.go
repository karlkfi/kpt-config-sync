package parse

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// updater mutates the most-recently-seen versions of objects stored in memory.
type updater struct {
	scope      declared.Scope
	resources  *declared.Resources
	remediator remediator.Interface
	applier    applier.Interface
}

func (u *updater) needToUpdateWatch() bool {
	return u.remediator.NeedsUpdate()
}

func (u *updater) managementConflict() bool {
	return u.remediator.ManagementConflict()
}

// declaredCRDs returns the list of CRDs which are present in the updater's
// declared resources.
func (u *updater) declaredCRDs() ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
	var crds []*v1beta1.CustomResourceDefinition
	for _, obj := range u.resources.Declarations() {
		if obj.GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
			continue
		}
		crd, err := clusterconfig.AsCRD(obj)
		if err != nil {
			return nil, err
		}
		crds = append(crds, crd)
	}
	return crds, nil
}

// update updates the declared resources in memory, applies the resources, and sets
// up the watches.
func (u *updater) update(ctx context.Context, cache *cacheForCommit) status.MultiError {
	var errs status.MultiError
	objs := filesystem.AsCoreObjects(cache.objsToApply)

	// Update the declared resources so that the Remediator immediately
	// starts enforcing the updated state.
	if !cache.resourceDeclSetUpdated {
		objs, err := u.resources.Update(ctx, objs)
		metrics.RecordDeclaredResources(ctx, len(objs))
		if err != nil {
			glog.Infof("Terminate the reconciliation (failed to update the declared resources): %v", err)
			return err
		}

		if cache.parserErrs == nil {
			cache.resourceDeclSetUpdated = true
		}
	}

	var gvks map[schema.GroupVersionKind]struct{}
	if cache.hasApplierResult {
		gvks = cache.applierResult
	} else {
		var applyErrs status.MultiError
		applyStart := time.Now()
		// TODO(b/168223472): This will show users a transient error if they apply a
		//  CRD + CR in the same commit. The caller should check for KNV2007, and
		//  retry without updating the respective status field as this is an expected
		//  path.
		gvks, applyErrs = u.applier.Apply(ctx, objs)
		metrics.RecordLastApplyAndDuration(ctx, metrics.StatusTagKey(applyErrs), cache.git.commit, applyStart)
		if applyErrs == nil && cache.parserErrs == nil {
			cache.setApplierResult(gvks)
		}
		errs = status.Append(errs, applyErrs)
	}

	// Update the Remediator's watches to start new ones and stop old ones.
	remediatorStart := time.Now()
	watchErrs := u.remediator.UpdateWatches(ctx, gvks)
	metrics.RecordWatchManagerUpdatesDuration(ctx, metrics.StatusTagKey(watchErrs), remediatorStart)
	errs = status.Append(errs, watchErrs)

	return errs
}
