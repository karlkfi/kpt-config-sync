package fake

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepoSyncObject initializes a RepoSync.
func RepoSyncObject(opts ...core.MetaMutator) *v1alpha1.RepoSync {
	result := &v1alpha1.RepoSync{
		ObjectMeta: metav1.ObjectMeta{
			Name: v1alpha1.RepoSyncName,
		},
		TypeMeta: ToTypeMeta(kinds.RepoSync()),
	}
	mutate(result, opts...)

	return result
}
