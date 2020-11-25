package parse

import (
	"context"
	"time"

	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/status"
)

// updater mutates the most-recently-seen versions of objects stored in memory.
type updater struct {
	scope      declared.Scope
	resources  *declared.Resources
	remediator remediator.Interface
	applier    applier.Interface
}

func (u *updater) needsUpdate() bool {
	return u.remediator.NeedsUpdate()
}

func (u *updater) update(ctx context.Context, objs []core.Object) status.MultiError {
	// First update the declared resources so that the Remediator immediately
	// starts enforcing the updated state.
	if err := u.resources.Update(objs); err != nil {
		return status.Append(nil, err)
	}
	metrics.RecordDeclaredResources(ctx, declared.ScopeName(u.scope), len(objs))

	// Then apply all new declared resources.
	// TODO(b/168223472): This will show users a transient error if they apply a
	//  CRD + CR in the same commit. The caller should check for KNV2007, and
	//  retry without updating the respective status field as this is an expected
	//  path.
	applyStart := time.Now()
	gvks, applyErrs := u.applier.Apply(ctx, objs)
	metrics.RecordLastApplyAndDuration(ctx, declared.ScopeName(u.scope), metrics.StatusTagKey(applyErrs), applyStart)

	// Finally update the Remediator's watches to start new ones and stop old
	// ones.
	remediatorStart := time.Now()
	watchErrs := u.remediator.UpdateWatches(ctx, gvks)
	metrics.RecordWatchManagerUpdateAndDuration(ctx, declared.ScopeName(u.scope), metrics.StatusTagKey(watchErrs), remediatorStart)

	return status.Append(applyErrs, watchErrs)
}
