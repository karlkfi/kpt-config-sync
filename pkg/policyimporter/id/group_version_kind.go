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
		group(gvk.GroupKind()), gvk.Version, gvk.Kind)
}

// printGroupKind returns a human-readable output for the GroupKind.
func printGroupKind(gvk schema.GroupKind) string {
	return fmt.Sprintf(
		"group:%[1]s\n"+
			"kind: %[2]s",
		group(gvk), gvk.Kind)
}

// group returns the empty string if gvk.Group is the empty string, otherwise prepends a space.
func group(gk schema.GroupKind) string {
	if gk.Group == "" {
		// Avoid unsightly whitespace if group is the empty string.
		return ""
	}
	// Prepends space to separate it from "group:"
	return " " + gk.Group
}
