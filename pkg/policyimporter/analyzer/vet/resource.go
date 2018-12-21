package vet

import (
	"fmt"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceAddr represents the necessary metadata for a user to find a Resource.
type ResourceAddr struct {
	// Source is the source file containing the Resource.
	Source string
	// Name is the metadata.name which the Resource declares.
	Name string
	// GVK is the Resource's Group/Version/Kind
	GVK schema.GroupVersionKind
}

// String implements Stringer
func (r ResourceAddr) String() string {
	return fmt.Sprintf("source: %[1]s\n"+
		"metadata.name: %[2]s\n"+
		"group: %[3]s\n"+
		"version: %[4]s\n"+
		"kind: %[5]s",
		r.Source, r.Name, r.GVK.Group, r.GVK.Version, r.GVK.Kind)
}

// ToResourceAddr converts an ast.FileObject to a Resource.
func ToResourceAddr(object ast.FileObject) ResourceAddr {
	return ResourceAddr{
		Source: object.Source,
		Name:   object.Name(),
		GVK:    object.GroupVersionKind(),
	}
}
