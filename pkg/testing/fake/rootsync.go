package fake

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RootSyncObject initializes a RootSync.
func RootSyncObject(opts ...core.MetaMutator) *v1alpha1.RootSync {
	result := &v1alpha1.RootSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.RootSyncName,
			Namespace: constants.ControllerNamespace,
		},
		TypeMeta: ToTypeMeta(kinds.RootSync()),
	}
	mutate(result, opts...)

	return result
}
