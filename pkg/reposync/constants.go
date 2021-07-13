package reposync

import (
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/declared"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectKey returns a key appropriate for fetching a RepoSync in the given
// namespace.
func ObjectKey(scope declared.Scope) client.ObjectKey {
	return client.ObjectKey{
		Namespace: string(scope),
		Name:      configsync.RepoSyncName,
	}
}
