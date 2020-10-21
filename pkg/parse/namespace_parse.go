package parse

import (
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/discovery"
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
func NewNamespace(fileReader filesystem.Reader, dc discovery.ServerResourcer, scope declared.Scope) *Namespace {
	return &Namespace{
		discoveryInterface: dc,
		parser:             filesystem.NewRawParser(fileReader, dc, string(scope)),
		scope:              scope,
	}
}

var _ filesystem.ConfigParser = &Namespace{}

// Parse implements filesystem.ConfigParser.
func (n Namespace) Parse(clusterName string, enableAPIServerChecks bool, getSyncedCRDs filesystem.GetSyncedCRDs, policyDir cmpath.Absolute, files []cmpath.Absolute) ([]core.Object, status.MultiError) {
	cos, err := n.parser.Parse(clusterName, enableAPIServerChecks, getSyncedCRDs, policyDir, files)
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

	var scoper discovery.Scoper
	scoper, _, err = filesystem.BuildScoper(n.discoveryInterface, enableAPIServerChecks, objs, nil, getSyncedCRDs)
	if err != nil {
		return nil, err
	}

	// We recreate this validator with every run as the set of available CRDs may
	// change between runs. The user may have either declared new CRDs in the root
	// repo, or they may have manually applied new ones.
	err = noClusterScopeValidator(scoper).Validate(objs)
	if err != nil {
		return nil, err
	}

	nsv := repositoryScopeVisitor(n.scope)
	err = nsv.Validate(objs)
	if err != nil {
		return nil, err
	}
	return cos, nil
}

// ReadClusterRegistryResources implements filesystem.ConfigParser.
func (n Namespace) ReadClusterRegistryResources(root cmpath.Absolute, files []cmpath.Absolute) []ast.FileObject {
	return n.parser.ReadClusterRegistryResources(root, files)
}
