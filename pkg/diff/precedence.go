package diff

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/syncer/differ"
)

// CanManage returns true if the given reconciler is allowed to manage the given
// resource.
func CanManage(reconciler declared.Scope, obj core.LabeledAndAnnotated) bool {
	if reconciler == declared.RootReconciler {
		// The root reconciler can always manage any resource.
		return true
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		// If the object somehow has no annotations, it is unmanaged and therefore
		// can be managed.
		return true
	}
	// TODO(b/166780454): Validate the resource's current Scope.
	manager, ok := annotations[v1alpha1.ResourceManagerKey]
	if !ok || !differ.ManagementEnabled(obj) {
		// Any reconciler can manage any unmanaged resource.
		return true
	}
	switch manager {
	case string(declared.RootReconciler):
		// Only the root reconciler can manage its own resources.
		return false
	default:
		// Ideally we would verify that the calling reconciler matches the annotated
		// manager. However we do not yet have a validating admission controller to
		// protect our annotations from being modified by users or controllers. A
		// user could block a non-root reconciler by modfiying the value of this
		// annotation to not match the proper reconciler.
		return true
	}
}
