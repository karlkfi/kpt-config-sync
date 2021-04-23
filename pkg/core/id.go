package core

import (
	"fmt"
	"strings"

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

// GKNN is used to set and verify the `configsync.gke.io/resource-id` annotation.
// Changing this function should be avoided, since it may
// introduce incompability across different Config Sync versions.
func GKNN(o client.Object) string {
	group := o.GetObjectKind().GroupVersionKind().Group
	kind := o.GetObjectKind().GroupVersionKind().Kind
	if o.GetNamespace() == "" {
		return fmt.Sprintf("%s_%s_%s", group, strings.ToLower(kind), o.GetName())
	}
	return fmt.Sprintf("%s_%s_%s_%s", group, strings.ToLower(kind), o.GetNamespace(), o.GetName())
}
