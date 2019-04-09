package id

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HierarchyConfig identifies a Group/Kind which has been declared in a HierarchyConfig in a Nomos repository.
// Unique so long as no single file illegally defines two Kinds of the same Group/Kind.
type HierarchyConfig interface {
	// Resource is the embedded interface providing path information to this HierarchyConfig.
	Resource
	// GroupKind returns the K8S Group/Kind the HierarchyConfig defines.
	GroupKind() schema.GroupKind
}
