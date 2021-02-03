package parse

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/reader"
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
	// cache tracks the progress made by the updater
	cache
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
		if s, err := reader.AsStruct(obj); err != nil {
			return nil, reader.ObjectParseError(obj, err)
		} else if crd, err := clusterconfig.AsCRD(s.(core.Object)); err != nil {
			return nil, err
		} else {
			crds = append(crds, crd)
		}
	}
	return crds, nil
}

// update updates the declared resources in memory, applies the resources, and sets
// up the watches.
func (u *updater) update(ctx context.Context, objs []core.Object) status.MultiError {
	var errs status.MultiError

	// Update the declared resources so that the Remediator immediately
	// starts enforcing the updated state.
	//
	// use `u.cache.resourceDeclSetUpdated` instead of `u.resourceDeclSetUpdated` here to
	// avoid a false-positive lint issue which does not go away by updating golangci/golangci-lint to v1.35.0:
	//   `resourceDeclSetUpdated` is unused (structcheck)
	if !u.cache.resourceDeclSetUpdated {
		err := u.resources.Update(objs)
		if err != nil {
			errs = status.Append(errs, err)
		} else {
			u.cache.resourceDeclSetUpdated = true
			metrics.RecordDeclaredResources(ctx, len(objs))
		}
	}

	var gvks map[schema.GroupVersionKind]struct{}
	if u.hasApplierResult {
		gvks = u.applierResult
	} else {
		var applyErrs status.MultiError
		applyStart := time.Now()
		// TODO(b/168223472): This will show users a transient error if they apply a
		//  CRD + CR in the same commit. The caller should check for KNV2007, and
		//  retry without updating the respective status field as this is an expected
		//  path.
		gvks, applyErrs = u.applier.Apply(ctx, objs)
		metrics.RecordLastApplyAndDuration(ctx, metrics.StatusTagKey(applyErrs), applyStart)
		if applyErrs == nil {
			u.setApplierResult(gvks)
		}
		errs = status.Append(errs, applyErrs)
	}

	// Update the Remediator's watches to start new ones and stop old ones.
	remediatorStart := time.Now()
	watchErrs := u.remediator.UpdateWatches(ctx, gvks)
	metrics.RecordWatchManagerUpdateAndDuration(ctx, metrics.StatusTagKey(watchErrs), remediatorStart)
	errs = status.Append(errs, watchErrs)

	return errs
}
