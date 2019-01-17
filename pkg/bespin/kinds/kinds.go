package kinds

import "k8s.io/apimachinery/pkg/runtime/schema"

// IAMPolicy returns the Group and Kind of IAMPolicies.
func IAMPolicy() schema.GroupKind {
	return schema.GroupKind{
		Group: "apiextensions.k8s.io",
		Kind:  "IAMPolicy",
	}
}
