package declared

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/syncer/differ"
)

// RootReconciler is a special constant for a reconciler which is running as the
// "root reconciler" (vs a namespace reconciler).
const RootReconciler = ":root"

// CanManage returns true if the given reconciler is allowed to manage the given
// resource.
func CanManage(reconciler string, obj core.LabeledAndAnnotated) bool {
	if reconciler == RootReconciler {
		// The root reconciler can always manage any resource.
		return true
	}
	annos := obj.GetAnnotations()
	if annos == nil {
		// If the object somehow has no annotations, it is unmanaged and therefore
		// can be managed.
		return true
	}
	manager, ok := annos[v1.ResourceManagerKey]
	if !ok || !differ.ManagementEnabled(obj) {
		// Any reconciler can manage any unmanaged resource.
		return true
	}
	switch manager {
	case RootReconciler:
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
