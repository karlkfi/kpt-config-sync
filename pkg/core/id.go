package core

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ID uniquely identifies a resource on an API Server.
type ID struct {
	schema.GroupKind
	client.ObjectKey
}

// IDOf converts an Object to its ID.
func IDOf(o Object) ID {
	return ID{
		GroupKind: o.GroupVersionKind().GroupKind(),
		ObjectKey: client.ObjectKey{Namespace: o.GetNamespace(), Name: o.GetName()},
	}
}

// String implements fmt.Stringer.
func (i ID) String() string {
	return fmt.Sprintf("%s, %s/%s", i.GroupKind.String(), i.Namespace, i.Name)
}

// IDOfRuntime is a convenience method for getting the ID of a runtime.Object.
func IDOfRuntime(o runtime.Object) (ID, error) {
	obj, err := ObjectOf(o)
	if err != nil {
		return ID{}, err
	}
	return IDOf(obj), nil
}

// IDOfUnstructured returns the ID of an unstructured object
func IDOfUnstructured(u unstructured.Unstructured) ID {
	return ID{
		GroupKind: u.GroupVersionKind().GroupKind(),
		ObjectKey: client.ObjectKey{
			Name:      u.GetName(),
			Namespace: u.GetNamespace(),
		},
	}
}
