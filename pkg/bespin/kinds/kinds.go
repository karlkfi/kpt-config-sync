package kinds

import "k8s.io/apimachinery/pkg/runtime/schema"

// IAMPolicy returns the Group and Kind of IAMPolicies.
func IAMPolicy() schema.GroupKind {
	return schema.GroupKind{
		Group: "apiextensions.k8s.io",
		Kind:  "IAMPolicy",
	}
}

// Organization returns the Group and Kind of Organizations.
func Organization() schema.GroupKind {
	return schema.GroupKind{
		Group: "bespin.dev",
		Kind:  "Organization",
	}
}

// Folder returns the Group and Kind of Folders.
func Folder() schema.GroupKind {
	return schema.GroupKind{
		Group: "bespin.dev",
		Kind:  "Folder",
	}
}

// Project returns the Group and Kind of Projects.
func Project() schema.GroupKind {
	return schema.GroupKind{
		Group: "bespin.dev",
		Kind:  "Project",
	}
}
