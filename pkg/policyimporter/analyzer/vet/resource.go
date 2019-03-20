package vet

import (
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
	"github.com/google/nomos/pkg/policyimporter/id"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// resourceID implements id.Resource in a minimal way. This enables directly instantiating errors for
// documentation or testing.
type resourceID struct {
	source           string
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

// Name implements Resource.
func (r resourceID) Name() string {
	return r.name
}

// GroupVersionKind implements Resource.
func (r resourceID) GroupVersionKind() schema.GroupVersionKind {
	return r.groupVersionKind
}
