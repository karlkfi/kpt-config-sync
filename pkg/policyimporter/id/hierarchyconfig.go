package id

import (
	"fmt"

	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HierarchyConfig identifies a Group/Kind which has been declared in a HierarchyConfig in a Nomos repository.
// Unique so long as no single file illegally defines two Kinds of the same Group/Kind.
type HierarchyConfig interface {
	// Sourced is the embedded interface providing path information to this HierarchyConfig.
	nomospath.Sourced
	// GroupKind returns the K8S Group/Kind the HierarchyConfig defines.
	GroupKind() schema.GroupKind
}

// PrintHierarchyConfig returns a human-readable output for the HierarchyConfig.
func PrintHierarchyConfig(c HierarchyConfig) string {
	return fmt.Sprintf("source: %[1]s\n"+
		"%[2]s",
		c.RelativeSlashPath(), printGroupKind(c.GroupKind()))
}
