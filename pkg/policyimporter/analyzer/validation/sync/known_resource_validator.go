package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/meta"
	"github.com/google/nomos/pkg/util/multierror"
)

// KnownResourceValidator ensures only Resources with definitions in the cluster are declared in Syncs.
type KnownResourceValidator struct {
	APIInfo *meta.APIInfo
}

// Validate adds errors for each unknown Resource Kind defined in Syncs.
func (v KnownResourceValidator) Validate(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	v.knownResourceValidator().Validate(objects, errorBuilder)
}

func (v KnownResourceValidator) knownResourceValidator() *validator {
	return &validator{
		validate: func(sync kindSync) error {
			gvk := sync.gvk
			if !isUnsupported(gvk) && !v.APIInfo.Exists(gvk) {
				return vet.UnknownResourceInSyncError{SyncPath: sync.sync.Source, ResourceType: gvk}
			}
			return nil
		},
	}
}
