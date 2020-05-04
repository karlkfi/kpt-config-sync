package fake

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NamespaceObject returns an initialized Namespace.
func NamespaceObject(name string, opts ...core.MetaMutator) *v1.Namespace {
	result := &v1.Namespace{TypeMeta: toTypeMeta(kinds.Namespace())}
	defaultMutate(result)
	mutate(result, core.Name(name))
	mutate(result, opts...)

	return result
}

// Namespace returns a Namespace FileObject with the passed opts.
//
// namespaceDir is the directory path within namespaces/ to the Namespace. Parses
//   namespacesDir to determine valid default metadata.Name.
func Namespace(dir string, opts ...core.MetaMutator) ast.FileObject {
	relative := cmpath.RelativeSlash(dir).Join(cmpath.RelativeSlash("namespace.yaml"))
	return NamespaceAtPath(relative.SlashPath(), opts...)
}

// NamespaceAtPath returns a Namespace at exactly the passed path.
func NamespaceAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	name := cmpath.RelativeSlash(path).Dir().Base()
	return FileObject(NamespaceObject(name, opts...), path)
}

// NamespaceUnstructured returns an unstructured Namespace FileObject with the passed opts.
func NamespaceUnstructured(dir string, opts ...core.MetaMutator) ast.FileObject {
	relative := cmpath.RelativeSlash(dir).Join(cmpath.RelativeSlash("namespace.yaml"))
	return NamespaceUnstructuredAtPath(relative.SlashPath(), opts...)
}

// NamespaceUnstructuredAtPath returns an unstructured Namespace at exactly the passed path.
func NamespaceUnstructuredAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	name := cmpath.RelativeSlash(path).Dir().Base()
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(kinds.Namespace())
	mutate(u, core.Name(name))
	mutate(u, opts...)
	return FileObject(u, path)
}
