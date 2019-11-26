package transform

import (
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// IsEphemeral returns true if the type should not be synced to the cluster.
func IsEphemeral(gvk schema.GroupVersionKind) bool {
	return gvk == kinds.NamespaceSelector()
}

// EphemeralResources returns the APIResourceLists of the ephemeral resources.
func EphemeralResources() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			GroupVersion: kinds.NamespaceSelector().GroupVersion().String(),
			APIResources: []metav1.APIResource{{Kind: kinds.NamespaceSelector().Kind, Namespaced: true}},
		},
	}
}
