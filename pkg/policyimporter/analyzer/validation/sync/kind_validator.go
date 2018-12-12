package sync

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
)

// KindValidator ensures that only supported Resource Kinds are declared in Syncs.
var KindValidator = &validator{
	validate: func(sync kindSync) error {
		if isUnsupported(sync.gvk) {
			return vet.UnsupportedResourceInSyncError{
				SyncPath:     sync.sync.Source,
				ResourceType: sync.gvk,
			}
		}
		return nil
	},
}
