package core

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// IDPath identifies a Kubernetes resource declared at a specific path in a
// Nomos repository.
type IDPath struct {
	ID
	id.Path
}

var _ id.Resource = &IDPath{}

// GetNamespace implements id.Resource.
func (i IDPath) GetNamespace() string {
	return i.Namespace
}

// GetName implements id.Resource.
func (i IDPath) GetName() string {
	return i.Name
}

// GroupVersionKind implements id.Resource.
func (i IDPath) GroupVersionKind() schema.GroupVersionKind {
	return i.GroupKind.WithVersion("")
}

// GetAnnotations implements id.Resource.
func (i IDPath) GetAnnotations() map[string]string {
	return map[string]string{
		v1.SourcePathAnnotationKey: i.SlashPath(),
	}
}
