package parse

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/vet"
)

// Namespace is a filesystem.ConfigParser that parses Namespace
// repositories.
//
// It wraps a filesystem.rawParser and adds a few additional validation steps.
type Namespace struct {
	discoveryInterface discovery.ServerResourcer
	parser             filesystem.ConfigParser
	scope              declared.Scope
}

// NewNamespace creates a new Namespace.
func NewNamespace(fileReader reader.Reader, dc discovery.ServerResourcer, scope declared.Scope) *Namespace {
	return &Namespace{
		discoveryInterface: dc,
		parser:             filesystem.NewRawParser(fileReader, dc, string(scope), scope),
		scope:              scope,
	}
}

var _ filesystem.ConfigParser = &Namespace{}

// Parse implements filesystem.ConfigParser.
func (n Namespace) Parse(clusterName string, enableAPIServerChecks bool, addCachedAPIResources vet.AddCachedAPIResourcesFn, getSyncedCRDs filesystem.GetSyncedCRDs, filePaths reader.FilePaths) ([]core.Object, status.MultiError) {
	cos, err := n.parser.Parse(clusterName, enableAPIServerChecks, addCachedAPIResources, getSyncedCRDs, filePaths)
	if err != nil {
		return nil, err
	}

	// Parse and generate a ResourceGroup from the Kptfile if it exists
	cos, e := AsResourceGroup(cos)
	if e != nil {
		err = status.Append(err, e)
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
