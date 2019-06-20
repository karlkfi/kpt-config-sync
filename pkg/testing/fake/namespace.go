package fake

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"k8s.io/api/core/v1"
)

// NamespaceObject returns an initialized Namespace.
func NamespaceObject(name string, opts ...object.MetaMutator) *v1.Namespace {
	result := &v1.Namespace{TypeMeta: toTypeMeta(kinds.Namespace())}
	defaultMutate(result)
	mutate(result, object.Name(name))
	mutate(result, opts...)

	return result
}

// Namespace returns a Namespace fileObject with the passed opts.
//
// namespaceDir is the directory path within namespaces/ to the Namespace. Parses
//   namespacesDir to determine valid default metadata.Name.
func Namespace(dir string, opts ...object.MetaMutator) ast.FileObject {
	relative := cmpath.FromSlash(dir).Join("namespace.yaml")
	name := relative.Dir().Base()

	return fileObject(NamespaceObject(name, opts...), relative.SlashPath())
}
