package diff

import (
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/syncer/differ"
)

// IsManager returns true if the given reconciler is the manager for the resource.
func IsManager(reconciler declared.Scope, obj core.LabeledAndAnnotated) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}
	manager, ok := annotations[v1alpha1.ResourceManagerKey]
	if !ok || !differ.ManagementEnabled(obj) {
		return false
	}
	return manager == string(reconciler)
}

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
	manager, ok := annotations[v1alpha1.ResourceManagerKey]
	if !ok || !differ.ManagementEnabled(obj) {
		// Any reconciler can manage any unmanaged resource.
		return true
	}
	if manager != "" {
		// Most objects have no manager, and ValidateScope will return an error in
		// this case. Explicitly checking for empty string means we don't do this
		// relatively expensive operation every time we're processing an object.
		if err := declared.ValidateScope(manager); err != nil {
			// We don't care if the actual object's manager declaration is invalid.
			// If it is and it's a managed object, we'll just overwrite it anyway.
			// If it isn't actually managed, we'll show this message every time the
			// object is updated - it is on the user to not mess with these annotations
			// if they don't want to see the error message.
			glog.Warningf("Invalid manager annotation %s=%q", v1alpha1.ResourceManagerKey, manager)
		}
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
