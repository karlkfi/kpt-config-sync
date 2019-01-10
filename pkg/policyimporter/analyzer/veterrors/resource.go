package veterrors

import (
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

// RelativeSlashPath implements Resource
func (r resourceID) RelativeSlashPath() string {
	return r.source
}

// Name implements Resource
func (r resourceID) Name() string {
	return r.name
}

// GroupVersionKind implements Resource
func (r resourceID) GroupVersionKind() schema.GroupVersionKind {
	return r.groupVersionKind
}

// syncID implements Sync in a minimal way. This enables directly instantiating errors for
// documentation or testing.
type syncID struct {
	source           string
	groupVersionKind schema.GroupVersionKind
}

var _ id.Sync = syncID{}

// RelativeSlashPath implements Sync
func (s syncID) RelativeSlashPath() string {
	return s.source
}

// GroupVersionKind implements Sync
func (s syncID) GroupVersionKind() schema.GroupVersionKind {
	return s.groupVersionKind
}
