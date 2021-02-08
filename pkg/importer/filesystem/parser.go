// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Parser reads files on disk and builds Nomos Config objects to be reconciled by the Syncer.
type Parser struct {
	reader            reader.Reader
	errOnUnknownKinds bool
	errors            status.MultiError
}

var _ ConfigParser = &Parser{}

// NewParser creates a new Parser using the given Reader and parser options.
func NewParser(reader reader.Reader, errOnUnknownKinds bool) *Parser {
	return &Parser{
		reader:            reader,
		errOnUnknownKinds: errOnUnknownKinds,
	}
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// clusterName is the spec.clusterName of the cluster's ConfigManagement.
// enableAPIServerChecks, if true, contacts the API Server if it is unable to
//   determine whether types are namespace- or cluster-scoped.
// getSyncedCRDs is a callback that returns the CRDs on the API Server.
// filePaths is the list of absolute file paths to parse and the absolute and
//   relative paths of the Nomos root.
// It is an error for any files not to be present.
func (p *Parser) Parse(clusterName string, syncedCRDs []*v1beta1.CustomResourceDefinition, buildScoper utildiscovery.BuildScoperFunc, filePaths reader.FilePaths) ([]ast.FileObject, status.MultiError) {
	p.errors = nil

	flatRoot := p.readObjects(filePaths)
	if p.errors != nil {
		return nil, p.errors
	}

	visitors := p.generateVisitors(filePaths.PolicyDir, flatRoot)
	fileObjects := p.hydrateRootAndFlatten(visitors, clusterName, syncedCRDs, buildScoper)

	return fileObjects, p.errors
}

// readObjects reads all objects in the repo and returns a FlatRoot holding all objects declared in
// manifests.
func (p *Parser) readObjects(filePaths reader.FilePaths) *ast.FlatRoot {
	return &ast.FlatRoot{
		SystemObjects:          p.readSystemResources(filePaths),
		ClusterRegistryObjects: p.ReadClusterRegistryResources(filePaths),
		ClusterObjects:         p.readClusterResources(filePaths),
		NamespaceObjects:       p.readNamespaceResources(filePaths),
	}
}

// generateVisitors creates the Visitors to use to hydrate and validate the root.
func (p *Parser) generateVisitors(policyDir cmpath.Relative, flatRoot *ast.FlatRoot) []ast.Visitor {
	visitors := []ast.Visitor{
		tree.NewSystemBuilderVisitor(flatRoot.SystemObjects),
		tree.NewClusterBuilderVisitor(flatRoot.ClusterObjects),
		tree.NewClusterRegistryBuilderVisitor(flatRoot.ClusterRegistryObjects),
		tree.NewBuilderVisitor(flatRoot.NamespaceObjects),
	}
	hierarchyConfigs, errs := extractHierarchyConfigs(flatRoot.SystemObjects)
	p.errors = status.Append(p.errors, errs)
	visitors = append(visitors, hierarchicalVisitors(policyDir, hierarchyConfigs)...)
	return visitors
}

// hydrateRootAndFlatten hydrates configuration into a fully-configured Root with the passed visitors.
func (p *Parser) hydrateRootAndFlatten(visitors []ast.Visitor, clusterName string, syncedCRDs []*v1beta1.CustomResourceDefinition, buildScoper utildiscovery.BuildScoperFunc) []ast.FileObject {
	astRoot := &ast.Root{
		ClusterName: clusterName,
	}

	root := p.runVisitors(astRoot, visitors)

	fileObjects := root.Flatten()

	declaredCRDs, err := customresources.GetCRDs(fileObjects)
	if err != nil {
		// We couldn't read the current CRDs, so we can't continue without showing a
		// lot of bogus errors to the user.
		p.errors = status.Append(p.errors, err)
		return nil
	}

	p.errors = status.Append(p.errors, nonhierarchical.CRDRemovalValidator(syncedCRDs, declaredCRDs).Validate(fileObjects))

	scoper, scoperErrs := buildScoper(declaredCRDs, fileObjects)
	if scoperErrs != nil {
		p.errors = status.Append(p.errors, scoperErrs)
		return nil
	}

	fileObjects, selErr := resolveHierarchicalSelectors(scoper, clusterName, fileObjects, p.errOnUnknownKinds)
	if selErr != nil {
		// Don't continue if selection failed; subsequent validation requires that selection succeeded.
		p.errors = status.Append(p.errors, selErr)
		return nil
	}

	p.errors = status.Append(p.errors, validation.NewTopLevelDirectoryValidator(scoper, p.errOnUnknownKinds).Validate(fileObjects))
	p.errors = status.Append(p.errors, hierarchyconfig.NewHierarchyConfigScopeValidator(scoper, p.errOnUnknownKinds).Validate(fileObjects))

	stdErrs := standardValidation(fileObjects)
	p.errors = status.Append(p.errors, stdErrs)

	fileObjects = selectors.AnnotateClusterName(clusterName, fileObjects)

	return fileObjects
}

func resolveHierarchicalSelectors(scoper utildiscovery.Scoper, clusterName string, fileObjects []ast.FileObject, enableAPIServerChecks bool) ([]ast.FileObject, status.MultiError) {
	// Validate and resolve cluster selectors.
	err := nonhierarchical.NewClusterSelectorAnnotationValidator().Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	err = validation.ClusterSelectorUniqueness.Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	fileObjects, err = selectors.ResolveClusterSelectors(clusterName, fileObjects)
	if err != nil {
		return nil, err
	}

	// Validate and resolve namespace selectors.
	err = nonhierarchical.NewNamespaceSelectorAnnotationValidator(scoper, enableAPIServerChecks).Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	err = validation.NamespaceSelectorUniqueness.Validate(fileObjects)
	if err != nil {
		return nil, err
	}

	fileObjects, err = selectors.ResolveHierarchicalNamespaceSelectors(fileObjects)
	if err != nil {
		return nil, err
	}

	return transform.RemoveEphemeralResources(fileObjects), nil
}

func (p *Parser) runVisitors(root *ast.Root, visitors []ast.Visitor) *ast.Root {
	for _, visitor := range visitors {
		if p.errors != nil && visitor.RequiresValidState() {
			return nil
		}
		root = root.Accept(visitor)
		p.errors = status.Append(p.errors, visitor.Error())
		if visitor.Fatal() {
			return nil
		}
	}
	return root
}

// filterTopDir returns the set of files contained in the top directory of root
//   along with the absolute and relative paths of root.
// Assumes all files are within root.
func filterTopDir(filePaths reader.FilePaths, topDir string) reader.FilePaths {
	rootSplits := filePaths.RootDir.Split()
	var result []cmpath.Absolute
	for _, f := range filePaths.Files {
		if f.Split()[len(rootSplits)] != topDir {
			continue
		}
		result = append(result, f)
	}
	return reader.FilePaths{
		RootDir:   filePaths.RootDir,
		PolicyDir: filePaths.PolicyDir,
		Files:     result,
	}
}

func (p *Parser) readSystemResources(filePaths reader.FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.SystemDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readNamespaceResources(filePaths reader.FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.NamespacesDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readClusterResources(filePaths reader.FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.ClusterDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

// ReadClusterRegistryResources reads the manifests declared in clusterregistry/.
func (p *Parser) ReadClusterRegistryResources(filePaths reader.FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.ClusterRegistryDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

// toInheritanceSpecs converts HierarchyConfigs to InheritanceSpecs. It also evaluates defaults so that later
// code doesn't have to.
func toInheritanceSpecs(configs []*v1.HierarchyConfig) map[schema.GroupKind]*transform.InheritanceSpec {
	specs := map[schema.GroupKind]*transform.InheritanceSpec{}
	for _, config := range configs {
		for _, r := range config.Spec.Resources {
			for _, k := range r.Kinds {
				gk := schema.GroupKind{Group: r.Group, Kind: k}
				var effectiveMode v1.HierarchyModeType
				if r.HierarchyMode == v1.HierarchyModeDefault {
					effectiveMode = v1.HierarchyModeInherit
				} else {
					effectiveMode = r.HierarchyMode
				}
				specs[gk] = &transform.InheritanceSpec{Mode: effectiveMode}
			}
		}
	}
	return specs
}
