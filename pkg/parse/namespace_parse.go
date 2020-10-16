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

// NamespaceParser is a filesystem.ConfigParser that parses Namespace
// repositories.
//
// It wraps a filesystem.rawParser and adds a few additional validation steps.
type NamespaceParser struct {
	discoveryInterface discovery.ServerResourcer
	parser             filesystem.ConfigParser
	scope              declared.Scope
}

// NewNamespaceParser creates a new NamespaceParser.
func NewNamespaceParser(fileReader filesystem.Reader, dc discovery.ServerResourcer, scope declared.Scope) *NamespaceParser {
	return &NamespaceParser{
		discoveryInterface: dc,
		parser:             filesystem.NewRawParser(fileReader, dc, string(scope)),
		scope:              scope,
	}
}

var _ filesystem.ConfigParser = &NamespaceParser{}

// Parse implements filesystem.ConfigParser.
func (p NamespaceParser) Parse(clusterName string, enableAPIServerChecks bool, getSyncedCRDs filesystem.GetSyncedCRDs, policyDir cmpath.Absolute, files []cmpath.Absolute) ([]core.Object, status.MultiError) {
	cos, err := p.parser.Parse(clusterName, enableAPIServerChecks,
		getSyncedCRDs, policyDir, files)
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

	scoper, _, err := filesystem.BuildScoper(p.discoveryInterface, true, objs, nil, getSyncedCRDs)
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

	nsv := repositoryScopeVisitor(p.scope)
	err = nsv.Validate(objs)
	if err != nil {
		return nil, err
	}
	return cos, nil
}

// ReadClusterRegistryResources implements filesystem.ConfigParser.
func (p NamespaceParser) ReadClusterRegistryResources(root cmpath.Absolute, files []cmpath.Absolute) []ast.FileObject {
	return p.parser.ReadClusterRegistryResources(root, files)
}
