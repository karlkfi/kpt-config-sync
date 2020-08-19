package difftest

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
)

// ManagedBy adds the annotation that a resource is managed by a particular
// Namespace's reconciler.
func ManagedBy(manager string) core.MetaMutator {
	return core.Annotation(v1.ResourceManagerKey, manager)
}

// ManagedByRoot indicates the resource is managed by the Root reconciler.
var ManagedByRoot = ManagedBy(declared.RootReconciler)
