package object

import "k8s.io/apimachinery/pkg/runtime/schema"

// GVKOpt modifies a schema.GroupVersionKind for testing.
type GVKOpt func(o *schema.GroupVersionKind)

// GVK modifies an existing schema.GroupVersionKind for testing.
func GVK(gvk schema.GroupVersionKind, opts ...GVKOpt) schema.GroupVersionKind {
	for _, opt := range opts {
		opt(&gvk)
	}
	return gvk
}

// Group replaces the Group of the GroupVersionKind with group.
func Group(group string) GVKOpt {
	return func(o *schema.GroupVersionKind) {
		o.Group = group
	}
}

// Version replaces the Group of the GroupVersionKind with version.
func Version(version string) GVKOpt {
	return func(o *schema.GroupVersionKind) {
		o.Version = version
	}
}

// Kind replaces the Group of the GroupVersionKind with kind.
func Kind(kind string) GVKOpt {
	return func(o *schema.GroupVersionKind) {
		o.Kind = kind
	}
}
