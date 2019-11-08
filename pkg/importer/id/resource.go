package id

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource identifies a Resource in a Nomos repository.
// Unique so long as no single file illegally declares two Resources of the same Name and Group/Version/Kind.
type Resource interface {
	// Path is the embedded interface providing path information to this Resource.
	Path
	// GetNamespace returns the namespace containing this resource.
	// If the resource is not namespaced, returns empty string.
	GetNamespace() string
	// GetName returns the metadata.name of the Resource.
	GetName() string
	// GroupVersionKind returns the K8S Group/Version/Kind of the Resource.
	GroupVersionKind() schema.GroupVersionKind
}

// PrintResource returns a human-readable output for the Resource.
func PrintResource(r Resource) string {
	var sb strings.Builder
	if r.SlashPath() != "" {
		sb.WriteString(fmt.Sprintf("source: %s\n", r.SlashPath()))
	}
	if r.GetNamespace() != "" {
		sb.WriteString(fmt.Sprintf("namespace: %s\n", r.GetNamespace()))
	}
	sb.WriteString(fmt.Sprintf("metadata.name:%s\n", name(r)))
	sb.WriteString(printGroupVersionKind(r.GroupVersionKind()))
	return sb.String()
}

// name returns the empty string if r.Name is the empty string, otherwise prepends a space.
func name(r Resource) string {
	if r.GetName() == "" {
		return ""
	}
	return " " + r.GetName()
}
