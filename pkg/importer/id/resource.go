package id

import (
	"fmt"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetSourceAnnotation returns the string value of the SourcePath Annotation.
// Returns empty string if unset or the object has no annotations.
func GetSourceAnnotation(obj client.Object) string {
	as := obj.GetAnnotations()
	if as == nil {
		return ""
	}
	return as[v1.SourcePathAnnotationKey]
}

// PrintResource returns a human-readable output for the Resource.
func PrintResource(r client.Object) string {
	var sb strings.Builder
	if sourcePath := GetSourceAnnotation(r); sourcePath != "" {
		sb.WriteString(fmt.Sprintf("source: %s\n", sourcePath))
	}
	if r.GetNamespace() != "" {
		sb.WriteString(fmt.Sprintf("namespace: %s\n", r.GetNamespace()))
	}
	sb.WriteString(fmt.Sprintf("metadata.name:%s\n", name(r)))
	sb.WriteString(printGroupVersionKind(r.GetObjectKind().GroupVersionKind()))
	return sb.String()
}

// name returns the empty string if r.Name is the empty string, otherwise prepends a space.
func name(r client.Object) string {
	if r.GetName() == "" {
		return ""
	}
	return " " + r.GetName()
}
