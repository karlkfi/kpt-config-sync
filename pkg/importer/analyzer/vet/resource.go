package vet

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// resourceID implements id.Resource in a minimal way. This enables directly instantiating errors for
// documentation or testing.
type resourceID struct {
	source           string
	namespace        string
	name             string
	groupVersionKind schema.GroupVersionKind
}

var _ id.Resource = resourceID{}

// SlashPath implements Resource.
func (r resourceID) SlashPath() string {
	return r.source
}

// OSPath implements Resource.
func (r resourceID) OSPath() string {
	return cmpath.FromSlash(r.source).OSPath()
}

// Dir implements Resource.
func (r resourceID) Dir() cmpath.Path {
	return cmpath.FromSlash(r.source).Dir()
}

// GetName implements Resource.
func (r resourceID) GetName() string {
	return r.name
}

// GetNamespace implements Resource.
func (r resourceID) GetNamespace() string {
	return r.namespace
}

// GroupVersionKind implements Resource.
func (r resourceID) GroupVersionKind() schema.GroupVersionKind {
	return r.groupVersionKind
}
