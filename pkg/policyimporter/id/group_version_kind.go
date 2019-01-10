package id

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// printGroupVersionKind returns a human-readable output for the GroupVersionKind.
func printGroupVersionKind(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf(
		"group: %[1]s\n"+
			"version: %[2]s\n"+
			"kind: %[3]s",
		gvk.Group, gvk.Version, gvk.Kind)
}
