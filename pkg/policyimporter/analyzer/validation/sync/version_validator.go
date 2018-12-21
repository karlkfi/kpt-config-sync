package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
)

// VersionValidator ensures that each Group/Kind with a Sync has exactly one version.
type VersionValidator struct {
	Objects []ast.FileObject
}

// Validate adds errors for each Group/Kind with multiple declarations.
func (v VersionValidator) Validate(errorBuilder *multierror.Builder) {
	syncKinds := make(map[groupKind][]vet.ResourceAddr)
	for _, sync := range kindSyncs(v.Objects) {
		gk := groupKind{group: sync.gvk.Group, kind: sync.gvk.Kind}
		syncKinds[gk] = append(syncKinds[gk], sync.sync)
	}

	for syncKind, duplicates := range syncKinds {
		if len(duplicates) > 1 {
			errorBuilder.Add(vet.DuplicateSyncGroupKindError{Group: syncKind.group, Kind: syncKind.kind, Syncs: duplicates})
		}
	}
}
