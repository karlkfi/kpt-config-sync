package id

import (
	"fmt"

	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Sync identifies a Kind which has been declared in a Sync in a Nomos repository.
// Unique so long as no single file illegally defines two Kinds of the same Group/Kind.
type Sync interface {
	// Sourced is the embedded interface providing path information to this Sync.
	nomospath.Sourced
	// GroupVersionKind returns the K8S Group/Version/Kind the Sync defines.
	GroupVersionKind() schema.GroupVersionKind
}

// PrintSync returns a human-readable output for the Sync.
func PrintSync(s Sync) string {
	return fmt.Sprintf("source: %[1]s\n"+
		"%[2]s",
		s.RelativeSlashPath(), printGroupVersionKind(s.GroupVersionKind()))
}
