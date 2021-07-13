package difftest

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/metadata"
)

// ManagedBy adds the annotation that a resource is managed by a particular
// Namespace's reconciler.
func ManagedBy(manager declared.Scope) core.MetaMutator {
	return core.Annotation(metadata.ResourceManagerKey, string(manager))
}

// ManagedByRoot indicates the resource is managed by the Root reconciler.
var ManagedByRoot = ManagedBy(declared.RootReconciler)
