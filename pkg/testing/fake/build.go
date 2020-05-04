package fake

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// defaultMutations are the standard Meta set on all fake objects. All can be overwritten with mutators.
//
// Annotations and Labels required when constructing any Object or else gomock will complain the nil
// and empty map are different. There is no other way to deal with this as the underlying
// implementations outside of our control handle empty vs nil maps inconsistently. Explicitly
// setting labels and annotations to empty map circumvents the issue.
var defaultMutations = []core.MetaMutator{
	core.Name("default-name"),
	core.Annotations(map[string]string{}),
	core.Labels(map[string]string{}),
}

func defaultMutate(object core.Object) {
	for _, m := range defaultMutations {
		m(object)
	}
}

func mutate(object core.Object, opts ...core.MetaMutator) {
	for _, m := range opts {
		m(object)
	}
}

// FileObject is a shorthand for converting to an ast.FileObject.
func FileObject(object core.Object, path string) ast.FileObject {
	return ast.NewFileObject(object, cmpath.RelativeSlash(path))
}

// UnstructuredObject initializes an unstructured.Unstructured.
func UnstructuredObject(gvk schema.GroupVersionKind, opts ...core.MetaMutator) *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.GetObjectKind().SetGroupVersionKind(gvk)

	defaultMutate(o)
	mutate(o, opts...)
	return o
}

// Unstructured initializes an Unstructured.
func Unstructured(gvk schema.GroupVersionKind, opts ...core.MetaMutator) ast.FileObject {
	return UnstructuredAtPath(gvk, "namespaces/obj.yaml", opts...)
}

// UnstructuredAtPath returns an Unstructured with the specified gvk.
func UnstructuredAtPath(gvk schema.GroupVersionKind, path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(UnstructuredObject(gvk, opts...), path)
}
