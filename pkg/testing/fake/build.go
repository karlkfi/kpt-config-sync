package fake

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/object"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// defaultMutations are the standard Meta set on all fake objects. All can be overwritten with mutators.
//
// Annotations and Labels required when constructing any Object or else gomock will complain the nil
// and empty map are different. There is no other way to deal with this as the underlying
// implementations outside of our control handle empty vs nil maps inconsistently. Explicitly
// setting labels and annotations to empty map circumvents the issue.
var defaultMutations = []object.MetaMutator{
	object.Name("default-name"),
	object.Annotations(map[string]string{}),
	object.Labels(map[string]string{}),
}

func defaultMutate(object v1.Object) {
	for _, m := range defaultMutations {
		m(object)
	}
}

func mutate(object v1.Object, opts ...object.MetaMutator) {
	for _, m := range opts {
		m(object)
	}
}

// FileObject is a shorthand for converting to an ast.FileObject.
func FileObject(object runtime.Object, path string) ast.FileObject {
	return ast.NewFileObject(object, cmpath.FromSlash(path))
}

// UnstructuredObject initializes an unstructured.Unstructured.
func UnstructuredObject(gvk schema.GroupVersionKind, opts ...object.MetaMutator) *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.GetObjectKind().SetGroupVersionKind(gvk)

	defaultMutate(o)
	mutate(o, opts...)
	return o
}

// Unstructured initializes an Unstructured.
func Unstructured(gvk schema.GroupVersionKind, opts ...object.MetaMutator) ast.FileObject {
	return UnstructuredAtPath(gvk, "namespaces/obj.yaml", opts...)
}

// UnstructuredAtPath returns an Unstructured with the specified gvk.
func UnstructuredAtPath(gvk schema.GroupVersionKind, path string, opts ...object.MetaMutator) ast.FileObject {
	return FileObject(UnstructuredObject(gvk, opts...), path)
}
