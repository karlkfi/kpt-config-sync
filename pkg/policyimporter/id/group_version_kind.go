package id

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// printGroupVersionKind returns a human-readable output for the GroupVersionKind.
func printGroupVersionKind(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf(
		"group:%[1]s\n"+
			"version: %[2]s\n"+
			"kind: %[3]s",
		group(gvk), gvk.Version, gvk.Kind)
}

// group returns the empty string if gvk.Group is the empty string, otherwise prepends a space.
func group(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		// Avoid unsightly whitespace if group is the empty string.
		return ""
	}
	// Prepends space to separate it from "group:"
	return " " + gvk.Group
}
