package fake

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
)

// RootSyncObject initializes a RootSync.
func RootSyncObject(opts ...core.MetaMutator) *v1alpha1.RootSync {
	result := &v1alpha1.RootSync{TypeMeta: ToTypeMeta(kinds.RootSync())}
	mutate(result, opts...)

	return result
}
