package fake

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
)

// RepoSyncObject initializes a RepoSync.
func RepoSyncObject(opts ...core.MetaMutator) *v1alpha1.RepoSync {
	result := &v1alpha1.RepoSync{TypeMeta: ToTypeMeta(kinds.RepoSync())}
	mutate(result, opts...)

	return result
}
