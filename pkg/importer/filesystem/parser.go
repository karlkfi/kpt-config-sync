// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/vet"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

func init() {
	// Add Nomos types to the Scheme used by asDefaultVersionedOrOriginal for
	// converting Unstructured to specific types.
	utilruntime.Must(apiextensions.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterregistry.AddToScheme(scheme.Scheme))
}

// Parser reads files on disk and builds Nomos Config objects to be reconciled by the Syncer.
type Parser struct {
	dc     utildiscovery.ServerResourcer
	reader Reader
	errors status.MultiError
}

var _ ConfigParser = &Parser{}

// NewParser creates a new Parser using the specified DiscoveryClient and parser options.
func NewParser(reader Reader, dc utildiscovery.ServerResourcer) *Parser {
	return &Parser{
		dc:     dc,
		reader: reader,
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
func (p *Parser) Parse(clusterName string, enableAPIServerChecks bool, addCachedAPIResources vet.AddCachedAPIResourcesFn, getSyncedCRDs GetSyncedCRDs, filePaths FilePaths) ([]core.Object, status.MultiError) {
	p.errors = nil

	flatRoot := p.readObjects(filePaths)
	if p.errors != nil {
		return nil, p.errors
	}

	visitors := generateVisitors(filePaths.PolicyDir, flatRoot)
	fileObjects := p.hydrateRootAndFlatten(visitors, clusterName, enableAPIServerChecks, addCachedAPIResources, getSyncedCRDs, filePaths.PolicyDir)

	return AsCoreObjects(fileObjects), p.errors
}

// readObjects reads all objects in the repo and returns a FlatRoot holding all objects declared in
// manifests.
func (p *Parser) readObjects(filePaths FilePaths) *ast.FlatRoot {
	return &ast.FlatRoot{
		SystemObjects:          p.readSystemResources(filePaths),
		ClusterRegistryObjects: p.ReadClusterRegistryResources(filePaths),
		ClusterObjects:         p.readClusterResources(filePaths),
		NamespaceObjects:       p.readNamespaceResources(filePaths),
	}
}

// generateVisitors creates the Visitors to use to hydrate and validate the root.
func generateVisitors(policyDir cmpath.Relative, flatRoot *ast.FlatRoot) []ast.Visitor {
	visitors := []ast.Visitor{
		tree.NewSystemBuilderVisitor(flatRoot.SystemObjects),
		tree.NewClusterBuilderVisitor(flatRoot.ClusterObjects),
		tree.NewClusterRegistryBuilderVisitor(flatRoot.ClusterRegistryObjects),
		tree.NewBuilderVisitor(flatRoot.NamespaceObjects),
	}
	hierarchyConfigs := extractHierarchyConfigs(flatRoot.SystemObjects)
	visitors = append(visitors, hierarchicalVisitors(policyDir, hierarchyConfigs)...)
	visitors = append(visitors, transform.NewSyncGenerator())
	return visitors
}

// hydrateRootAndFlatten hydrates configuration into a fully-configured Root with the passed visitors.
func (p *Parser) hydrateRootAndFlatten(visitors []ast.Visitor, clusterName string, enableAPIServerChecks bool, addCachedAPIResources vet.AddCachedAPIResourcesFn, getSyncedCRDs GetSyncedCRDs, policyDir cmpath.Relative) []ast.FileObject {
	astRoot := &ast.Root{
		ClusterName: clusterName,
	}

	root := p.runVisitors(astRoot, visitors)

	fileObjects := root.Flatten()

	declaredCRDs, err := customresources.GetCRDs(fileObjects)
	if err != nil {
		// We couldn't read the CRDs, so we can't continue without showing a lot of
		// bogus errors to the user.
		p.errors = status.Append(p.errors, err)
		return nil
	}

	scoper, syncedCRDs, scoperErrs := BuildScoper(p.dc, enableAPIServerChecks, addCachedAPIResources, fileObjects, declaredCRDs, getSyncedCRDs)
	if scoperErrs != nil {
		p.errors = status.Append(p.errors, scoperErrs)
		return nil
	}
	p.errors = status.Append(p.errors, nonhierarchical.CRDRemovalValidator(syncedCRDs, declaredCRDs).Validate(fileObjects))

	fileObjects, selErr := resolveHierarchicalSelectors(scoper, clusterName, fileObjects, enableAPIServerChecks)
	if selErr != nil {
		// Don't continue if selection failed; subsequent validation requires that selection succeeded.
		p.errors = status.Append(p.errors, selErr)
		return nil
	}

	p.errors = status.Append(p.errors, validation.NewTopLevelDirectoryValidator(scoper).Validate(fileObjects))
	p.errors = status.Append(p.errors, hierarchyconfig.NewHierarchyConfigScopeValidator(scoper).Validate(fileObjects))

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
func filterTopDir(filePaths FilePaths, topDir string) FilePaths {
	rootSplits := filePaths.RootDir.Split()
	var result []cmpath.Absolute
	for _, f := range filePaths.Files {
		if f.Split()[len(rootSplits)] != topDir {
			continue
		}
		result = append(result, f)
	}
	return FilePaths{filePaths.RootDir, filePaths.PolicyDir, result}
}

func (p *Parser) readSystemResources(filePaths FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.SystemDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readNamespaceResources(filePaths FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.NamespacesDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readClusterResources(filePaths FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.ClusterDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

// ReadClusterRegistryResources reads the manifests declared in clusterregistry/.
func (p *Parser) ReadClusterRegistryResources(filePaths FilePaths) []ast.FileObject {
	result, errs := p.reader.Read(filterTopDir(filePaths, repo.ClusterRegistryDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

func hasStatusField(u runtime.Unstructured) bool {
	// The following call will only error out if the UnstructuredContent returns something that is not a map.
	// This has already been verified upstream.
	m, ok, err := unstructured.NestedFieldNoCopy(u.UnstructuredContent(), "status")
	if err != nil {
		// This should never happen!!!
		glog.Errorf("unexpected error retrieving status field: %v:\n%v", err, u)
	}
	return ok && m != nil && len(m.(map[string]interface{})) != 0
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
