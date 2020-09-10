package parse

import (
	"context"

	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/remediator"
	"github.com/google/nomos/pkg/status"
)

// updater mutates the most-recently-seen versions of objects stored in memory.
type updater struct {
	remediator remediator.Interface
	applier    applier.Interface
}

func (u *updater) update(ctx context.Context, objs []core.Object) status.MultiError {
	// The Remediator MUST be updated before the applier.
	//
	// The main reason for this is to avoid a race condition where:
	// 1. the first resource of a GVK is added to Git
	// 2. the applier writes that resource to the cluster
	// 3. a user or controller modifies that resource
	// 4. the remediator establishes a new watch for that GVK

	// TODO(b/168223472): This will show users a transient error if they apply a
	//  CRD + CR in the same commit. The caller should check for KNV2007, and
	//  retry without updating the respective status field as this is an expected
	//  path.
	watched, updateErrs := u.remediator.Update(objs)
	applyErrs := u.applier.Apply(ctx, watched, objs)

	return status.Append(updateErrs, applyErrs)
}
