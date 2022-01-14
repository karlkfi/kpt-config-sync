package rootsync

import (
	"github.com/google/nomos/pkg/api/configsync"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectKey returns a key appropriate for fetching a RootSync.
func ObjectKey(name string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: configsync.ControllerNamespace,
		Name:      name,
	}
}
