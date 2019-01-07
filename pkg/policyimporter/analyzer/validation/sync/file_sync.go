package sync

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FileSync extends v1alpha1.Sync to include the path to the file in the repo.
type FileSync struct {
	*v1alpha1.Sync
	// Source is the OS-agnostic slash-separated path to the source file from the root.
	source string
}

var _ veterrors.SyncID = FileSync{}

// NewFileSync creates a new FileSync from a Sync Resource and the source file declearing the Sync.
func NewFileSync(sync *v1alpha1.Sync, source string) FileSync {
	return FileSync{Sync: sync, source: source}
}

// Source implements vet.ResourceID
func (s FileSync) Source() string {
	return s.source
}

// flatten returns a list of all GroupVersionKinds defined in the Sync and their hierarchy modes.
func (s FileSync) flatten() []FileGroupVersionKindHierarchySync {
	var result []FileGroupVersionKindHierarchySync
	for _, group := range s.Spec.Groups {
		for _, kind := range group.Kinds {
			for _, version := range kind.Versions {
				result = append(result, FileGroupVersionKindHierarchySync{
					groupVersionKind: schema.GroupVersionKind{Group: group.Group, Version: version.Version, Kind: kind.Kind},
					HierarchyMode:    kind.HierarchyMode,
					source:           s.source,
				})
			}
		}
	}
	return result
}

// FileGroupVersionKindHierarchySync Identifies a Group/Version/Kind definition in a Sync.
// This is not unique if the same Sync Resource defines the multiple of the same Group/Kind.
type FileGroupVersionKindHierarchySync struct {
	// GroupVersionKind is the Group/Version/Kind which the Sync defined.
	groupVersionKind schema.GroupVersionKind
	// HierarchyMode is the hierarchy mode which the Sync defined for the Kind.
	HierarchyMode v1alpha1.HierarchyModeType
	// Source is the OS-agnostic slash-separated path to the source file from the root.
	source string
}

// Source implements vet.SyncID
func (s FileGroupVersionKindHierarchySync) Source() string {
	return s.source
}

// GroupVersionKind implements vet.SyncID
func (s FileGroupVersionKindHierarchySync) GroupVersionKind() schema.GroupVersionKind {
	return s.groupVersionKind
}
