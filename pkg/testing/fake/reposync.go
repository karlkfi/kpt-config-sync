package fake

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
)

// RepoSyncObject initializes a RepoSync.
func RepoSyncObject(opts ...core.MetaMutator) *v1.RepoSync {
	result := &v1.RepoSync{TypeMeta: toTypeMeta(kinds.RepoSync())}
	mutate(result, opts...)

	return result
}
