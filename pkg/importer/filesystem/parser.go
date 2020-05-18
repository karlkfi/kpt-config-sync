// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	"github.com/golang/glog"
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
	"github.com/google/nomos/pkg/status"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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
	utilruntime.Must(v1beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterregistry.AddToScheme(scheme.Scheme))
}

// Parser reads files on disk and builds Nomos Config objects to be reconciled by the Syncer.
type Parser struct {
	clientGetter utildiscovery.ClientGetter
	reader       Reader
	errors       status.MultiError
}

var _ ConfigParser = &Parser{}

// NewParser creates a new Parser using the specified RESTClientGetter and parser options.
func NewParser(reader Reader, c utildiscovery.ClientGetter) *Parser {
	return &Parser{
		clientGetter: c,
		reader:       reader,
	}
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// clusterName is the spec.clusterName of the cluster's ConfigManagement.
// enableAPIServerChecks, if true, contacts the API Server if it is unable to
//   determine whether types are namespace- or cluster-scoped.
// getSyncedCRDs is a callback that returns the CRDs on the API Server.
// policyDir is the absolute root path of the policies.
// files is the complete list of absolute file paths to parse.
//   It is an error for any files not to be present.
func (p *Parser) Parse(
	clusterName string,
	enableAPIServerChecks bool,
	getSyncedCRDs GetSyncedCRDs,
	policyDir cmpath.Absolute,
	files []cmpath.Absolute,
) ([]ast.FileObject, status.MultiError) {
	p.errors = nil

	flatRoot := p.readObjects(policyDir, files)
	crds, err := customresources.GetCRDs(flatRoot.ClusterObjects)
	p.errors = status.Append(p.errors, err)
	if p.errors != nil {
		return nil, p.errors
	}

	visitors := p.generateVisitors(flatRoot, crds)
	if p.errors != nil {
		return nil, p.errors
	}
	fileObjects := p.hydrateRootAndFlatten(visitors, clusterName, enableAPIServerChecks, getSyncedCRDs)

	return fileObjects, p.errors
}

// readObjects reads all objects in the repo and returns a FlatRoot holding all objects declared in
// manifests.
func (p *Parser) readObjects(root cmpath.Absolute, files []cmpath.Absolute) *ast.FlatRoot {
	return &ast.FlatRoot{
		SystemObjects:          p.readSystemResources(root, files),
		ClusterRegistryObjects: p.ReadClusterRegistryResources(root, files),
		ClusterObjects:         p.readClusterResources(root, files),
		NamespaceObjects:       p.readNamespaceResources(root, files),
	}
}

// generateVisitors creates the Visitors to use to hydrate and validate the root.
func (p *Parser) generateVisitors(
	flatRoot *ast.FlatRoot,
	crds []*v1beta1.CustomResourceDefinition,
) []ast.Visitor {
	visitors := []ast.Visitor{
		tree.NewSystemBuilderVisitor(flatRoot.SystemObjects),
		tree.NewClusterBuilderVisitor(flatRoot.ClusterObjects),
		tree.NewClusterRegistryBuilderVisitor(flatRoot.ClusterRegistryObjects),
		tree.NewBuilderVisitor(flatRoot.NamespaceObjects),
	}
	hierarchyConfigs := extractHierarchyConfigs(flatRoot.SystemObjects)
	visitors = append(visitors, hierarchicalVisitors(hierarchyConfigs)...)
	visitors = append(visitors, transform.NewSyncGenerator())
	return visitors
}

// hydrateRootAndFlatten hydrates configuration into a fully-configured Root with the passed visitors.
func (p *Parser) hydrateRootAndFlatten(visitors []ast.Visitor, clusterName string, enableAPIServerChecks bool, getSyncedCRDs GetSyncedCRDs) []ast.FileObject {
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

	// We can't continue with hierarchical logic processing if there is any issue
	// establishing the scope of all declared resources.
	scoper, syncedCRDs, scoperErrs := buildScoper(p.clientGetter, enableAPIServerChecks, fileObjects, declaredCRDs, getSyncedCRDs)
	if scoperErrs != nil {
		p.errors = status.Append(p.errors, err)
		return nil
	}
	p.errors = status.Append(p.errors, nonhierarchical.CRDRemovalValidator(syncedCRDs, declaredCRDs).Validate(fileObjects))

	fileObjects, selErr := resolveHierarchicalSelectors(scoper, clusterName, fileObjects, enableAPIServerChecks)
	if selErr != nil {
		// Don't continue if selection failed; subsequent validation requires that selection succeeded.
		p.errors = status.Append(p.errors, selErr)
		return nil
	}

	p.errors = status.Append(p.errors, validation.NewTopLevelDirectoryValidator(scoper, enableAPIServerChecks).Validate(fileObjects))
	p.errors = status.Append(p.errors, hierarchyconfig.NewHierarchyConfigScopeValidator(scoper, enableAPIServerChecks).Validate(fileObjects))

	stdErrs := standardValidation(fileObjects)
	p.errors = status.Append(p.errors, stdErrs)

	fileObjects = selectors.AnnotateClusterName(clusterName, fileObjects)

	return fileObjects
}

func resolveHierarchicalSelectors(scoper utildiscovery.Scoper, clusterName string, fileObjects []ast.FileObject, enableAPIServerChecks bool) ([]ast.FileObject, status.MultiError) {
	annErr := nonhierarchical.NewSelectorAnnotationValidator(scoper, enableAPIServerChecks).Validate(fileObjects)
	if annErr != nil {
		return nil, annErr
	}

	csuErr := validation.ClusterSelectorUniqueness.Validate(fileObjects)
	if csuErr != nil {
		return nil, csuErr
	}

	fileObjects, csErr := selectors.ResolveClusterSelectors(clusterName, fileObjects)
	if csErr != nil {
		return nil, csErr
	}

	nsuErr := validation.NamespaceSelectorUniqueness.Validate(fileObjects)
	if nsuErr != nil {
		return nil, nsuErr
	}

	fileObjects, nsErr := selectors.ResolveHierarchicalNamespaceSelectors(fileObjects)
	if nsErr != nil {
		return nil, nsErr
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

// filterTopDir returns the set of file contained in the top directory of root.
// Assumes all files are within root.
func filterTopDir(root cmpath.Absolute, files []cmpath.Absolute, topDir string) []cmpath.Absolute {
	rootSplits := root.Split()
	var result []cmpath.Absolute
	for _, f := range files {
		if f.Split()[len(rootSplits)] != topDir {
			continue
		}
		result = append(result, f)
	}
	return result
}

func (p *Parser) readSystemResources(root cmpath.Absolute, files []cmpath.Absolute) []ast.FileObject {
	result, errs := p.reader.Read(root, filterTopDir(root, files, repo.SystemDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readNamespaceResources(root cmpath.Absolute, files []cmpath.Absolute, crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	result, errs := p.reader.Read(root, filterTopDir(root, files, repo.NamespacesDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readClusterResources(root cmpath.Absolute, files []cmpath.Absolute, crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	result, errs := p.reader.Read(root, filterTopDir(root, files, repo.ClusterDir))
	p.errors = status.Append(p.errors, errs)
	return result
}

// ReadClusterRegistryResources reads the manifests declared in clusterregistry/.
func (p *Parser) ReadClusterRegistryResources(root cmpath.Absolute, files []cmpath.Absolute) []ast.FileObject {
	result, errs := p.reader.Read(root, filterTopDir(root, files, repo.ClusterRegistryDir))
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
