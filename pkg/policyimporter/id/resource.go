package id

import (
	"fmt"

	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource identifies a Resource in a Nomos repository.
// Unique so long as no single file illegally declares two Resources of the same Name and Group/Version/Kind.
type Resource interface {
	// Sourced is the embedded interface providing path information to this Resource.
	nomospath.Sourced
	// Name returns the metadata.name of the Resource.
	Name() string
	// GroupVersionKind returns the K8S Group/Version/Kind of the Resource.
	GroupVersionKind() schema.GroupVersionKind
}

// PrintResource returns a human-readable output for the Resource.
func PrintResource(r Resource) string {
	return fmt.Sprintf("source: %[1]s\n"+
		"metadata.name: %[2]s\n"+
		"%[3]s",
		r.RelativeSlashPath(), r.Name(), printGroupVersionKind(r.GroupVersionKind()))
}
