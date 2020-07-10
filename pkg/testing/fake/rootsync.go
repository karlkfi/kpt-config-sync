package fake

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
)

// RootSyncObject initializes a RootSync.
func RootSyncObject(opts ...core.MetaMutator) *v1.RootSync {
	result := &v1.RootSync{TypeMeta: toTypeMeta(kinds.RootSync())}
	mutate(result, opts...)

	return result
}
