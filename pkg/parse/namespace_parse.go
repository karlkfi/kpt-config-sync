package parse

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// Namespace is a filesystem.ConfigParser that parses Namespace
// repositories.
//
// It wraps a filesystem.rawParser and adds a few additional validation steps.
type Namespace struct {
	parser filesystem.ConfigParser
	scope  declared.Scope
}

// NewNamespace creates a new Namespace.
func NewNamespace(fileReader reader.Reader, errOnUnknown bool, scope declared.Scope) *Namespace {
	return &Namespace{
		parser: filesystem.NewRawParser(fileReader, errOnUnknown, string(scope), scope),
		scope:  scope,
	}
}

var _ filesystem.ConfigParser = &Namespace{}

// Parse implements filesystem.ConfigParser.
func (n Namespace) Parse(clusterName string, syncedCRDs []*v1beta1.CustomResourceDefinition, buildScoper discovery.BuildScoperFunc, filePaths reader.FilePaths) ([]core.Object, status.MultiError) {
	cos, err := n.parser.Parse(clusterName, syncedCRDs, buildScoper, filePaths)
	if err != nil {
		return nil, err
	}

	objs := filesystem.AsFileObjects(cos)

	nsv := repositoryScopeVisitor(n.scope)
	err = nsv.Validate(objs)
	if err != nil {
		return nil, err
	}
	return cos, nil
}

// ReadClusterRegistryResources implements filesystem.ConfigParser.
func (n Namespace) ReadClusterRegistryResources(filePaths reader.FilePaths) []ast.FileObject {
	return n.parser.ReadClusterRegistryResources(filePaths)
}
