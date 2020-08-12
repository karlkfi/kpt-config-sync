package id

import (
	"fmt"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Annotated has Kubernetes annotations.
type Annotated interface {
	GetAnnotations() map[string]string
}

// Resource identifies a Resource in a Nomos repository.
// Unique so long as no single file illegally declares two Resources of the same Name and Group/Version/Kind.
type Resource interface {
	// GetAnnotations contains the source path annotation, if present.
	GetAnnotations() map[string]string
	// GetNamespace returns the namespace containing this resource.
	// If the resource is not namespaced, returns empty string.
	GetNamespace() string
	// GetName returns the metadata.name of the Resource.
	GetName() string
	// GroupVersionKind returns the K8S Group/Version/Kind of the Resource.
	GroupVersionKind() schema.GroupVersionKind
}

// GetSourceAnnotation returns the string value of the SourcePath Annotation.
// Returns empty string if unset or the object has no annotations.
func GetSourceAnnotation(obj Annotated) string {
	as := obj.GetAnnotations()
	if as == nil {
		return ""
	}
	return as[v1.SourcePathAnnotationKey]
}

// PrintResource returns a human-readable output for the Resource.
func PrintResource(r Resource) string {
	var sb strings.Builder
	if sourcePath := GetSourceAnnotation(r); sourcePath != "" {
		sb.WriteString(fmt.Sprintf("source: %s\n", sourcePath))
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
