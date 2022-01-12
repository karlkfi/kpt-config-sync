package fake

import (
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepoSyncObjectV1Alpha1 initializes a RepoSync with version v1alpha1.
func RepoSyncObjectV1Alpha1(opts ...core.MetaMutator) *v1alpha1.RepoSync {
	result := &v1alpha1.RepoSync{
		ObjectMeta: metav1.ObjectMeta{
			Name: configsync.RepoSyncName,
		},
		TypeMeta: ToTypeMeta(kinds.RepoSyncV1Alpha1()),
	}
	mutate(result, opts...)

	return result
}

// RepoSyncObjectV1Beta1 initializes a RepoSync with version v1beta1.
func RepoSyncObjectV1Beta1(opts ...core.MetaMutator) *v1beta1.RepoSync {
	result := &v1beta1.RepoSync{
		ObjectMeta: metav1.ObjectMeta{
			Name: configsync.RepoSyncName,
		},
		TypeMeta: ToTypeMeta(kinds.RepoSyncV1Beta1()),
	}
	mutate(result, opts...)

	return result
}
