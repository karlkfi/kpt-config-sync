package fake

import (
	"github.com/google/nomos/pkg/core"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OwnerReference adds an OwnerReference to the passed Object.
// Panics if the type does not define metadata.ownerReferences.
func OwnerReference(name string, gvk schema.GroupVersionKind) core.MetaMutator {
	return func(o client.Object) {
		owners := o.GetOwnerReferences()
		owners = append(owners, v1.OwnerReference{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
			Name:       name,
		})
		o.SetOwnerReferences(owners)
	}
}
