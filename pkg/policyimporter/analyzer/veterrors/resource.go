package veterrors

import (
	"fmt"

	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceID identifies a Resource in a Nomos repository.
// Unique so long as no single file illegally declares two Resources of the same Name and Group/Version/Kind.
type ResourceID interface {
	nomospath.Sourced
	// Name returns the metadata.name of the Resource.
	Name() string
	// GroupVersionKind returns the K8S Group/Version/Kind of the Resource.
	GroupVersionKind() schema.GroupVersionKind
}

// String implements Stringer
func printResourceID(r ResourceID) string {
	return fmt.Sprintf("source: %[1]s\n"+
		"metadata.name: %[2]s\n"+
		"%[3]s",
		r.RelativeSlashPath(), r.Name(), printGroupVersionKind(r.GroupVersionKind()))
}

// String implements Stringer
func printGroupVersionKind(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf(
		"group: %[1]s\n"+
			"version: %[2]s\n"+
			"kind: %[3]s",
		gvk.Group, gvk.Version, gvk.Kind)
}

// resourceID implements ResourceID in a minimal way. This enables directly instantiating errors for
// documentation or testing.
type resourceID struct {
	source           string
	name             string
	groupVersionKind schema.GroupVersionKind
}

var _ ResourceID = resourceID{}

// RelativeSlashPath implements ResourceID
func (r resourceID) RelativeSlashPath() string {
	return r.source
}

// Name implements ResourceID
func (r resourceID) Name() string {
	return r.name
}

// GroupVersionKind implements ResourceID
func (r resourceID) GroupVersionKind() schema.GroupVersionKind {
	return r.groupVersionKind
}

// SyncID identifies a Kind which has been declared in a Sync in a Nomos repository.
// Unique so long as no single file illegally defines two Kinds of the same Group/Kind.
type SyncID interface {
	nomospath.Sourced
	// GroupVersionKind returns the K8S Group/Version/Kind the Sync defines.
	GroupVersionKind() schema.GroupVersionKind
}

func printSyncID(s SyncID) string {
	return fmt.Sprintf("source: %[1]s\n"+
		"%[2]s",
		s.RelativeSlashPath(), printGroupVersionKind(s.GroupVersionKind()))
}

// syncID implements SyncID in a minimal way. This enables directly instantiating errors for
// documentation or testing.
type syncID struct {
	source           string
	groupVersionKind schema.GroupVersionKind
}

var _ SyncID = syncID{}

// RelativeSlashPath implements SyncID
func (s syncID) RelativeSlashPath() string {
	return s.source
}

// GroupVersionKind implements SyncID
func (s syncID) GroupVersionKind() schema.GroupVersionKind {
	return s.groupVersionKind
}
