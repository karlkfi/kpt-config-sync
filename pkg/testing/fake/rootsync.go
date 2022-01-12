package fake

import (
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RootSyncObjectV1Alpha1 initializes a RootSync.
func RootSyncObjectV1Alpha1(opts ...core.MetaMutator) *v1alpha1.RootSync {
	result := &v1alpha1.RootSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configsync.RootSyncName,
			Namespace: configsync.ControllerNamespace,
		},
		TypeMeta: ToTypeMeta(kinds.RootSyncV1Alpha1()),
	}
	mutate(result, opts...)

	return result
}

// RootSyncObjectV1Beta1 initializes a RootSync with version v1beta1.
func RootSyncObjectV1Beta1(opts ...core.MetaMutator) *v1beta1.RootSync {
	result := &v1beta1.RootSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configsync.RootSyncName,
			Namespace: configsync.ControllerNamespace,
		},
		TypeMeta: ToTypeMeta(kinds.RootSyncV1Beta1()),
	}
	mutate(result, opts...)

	return result
}
