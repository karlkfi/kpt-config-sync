package fake

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/api/core/v1"
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
	relative := cmpath.FromSlash(dir).Join("namespace.yaml")
	return NamespaceAtPath(relative.SlashPath(), opts...)
}

// NamespaceAtPath returns a Namespace at exactly the passed path.
func NamespaceAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	name := cmpath.FromSlash(path).Dir().Base()
	return FileObject(NamespaceObject(name, opts...), path)
}
