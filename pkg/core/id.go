package core

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ID uniquely identifies a resource on an API Server.
type ID struct {
	schema.GroupKind
	client.ObjectKey
}

// IDOf converts an Object to its ID.
func IDOf(o client.Object) ID {
	return ID{
		GroupKind: o.GetObjectKind().GroupVersionKind().GroupKind(),
		ObjectKey: client.ObjectKey{Namespace: o.GetNamespace(), Name: o.GetName()},
	}
}

// String implements fmt.Stringer.
func (i ID) String() string {
	return fmt.Sprintf("%s, %s/%s", i.GroupKind.String(), i.Namespace, i.Name)
}
