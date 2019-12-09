// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	"github.com/davecgh/go-spew/spew"
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
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/clusterconfig"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	opts         ParserOpt
	clientGetter utildiscovery.ClientGetter
	reader       Reader
	errors       status.MultiError
}

var _ ConfigParser = &Parser{}

// ParserOpt has often customizes the behavior of Parser.Parse.
type ParserOpt struct {
	// Extension is the ParserConfig object that the parser will consume for configuring various
	// aspects of the execution (see ParserConfig).
	Extension ParserConfig
	// RootPath is the file path to parse as GRoot.
	RootPath cmpath.Root
}

// NewParser creates a new Parser using the specified RESTClientGetter and parser options.
func NewParser(c utildiscovery.ClientGetter, opts ParserOpt) *Parser {
	p := &Parser{
		clientGetter: c,
		reader:       &FileReader{},
		opts:         opts,
	}
	return p
}

// Errors returns the errors the Parser has encountered so far.
func (p *Parser) Errors() status.MultiError {
	return p.errors
}

// ReadObjects reads all objects in the repo and returns a FlatRoot holding all objects declared in
// manifests.
func (p *Parser) ReadObjects() *ast.FlatRoot {
	return &ast.FlatRoot{
		SystemObjects:          p.readSystemResources(),
		ClusterRegistryObjects: p.ReadClusterRegistryResources(),
		ClusterObjects:         p.readClusterResources(),
		NamespaceObjects:       p.readNamespaceResources(),
	}
}

// GenerateVisitors creates the Visitors to use to hydrate and validate the root.
func (p *Parser) GenerateVisitors(
	flatRoot *ast.FlatRoot,
	currentConfigs *namespaceconfig.AllConfigs,
	crds []*v1beta1.CustomResourceDefinition,
) []ast.Visitor {
	visitors := []ast.Visitor{
		tree.NewSystemBuilderVisitor(flatRoot.SystemObjects),
		tree.NewClusterBuilderVisitor(flatRoot.ClusterObjects),
		tree.NewClusterRegistryBuilderVisitor(flatRoot.ClusterRegistryObjects),
		tree.NewBuilderVisitor(flatRoot.NamespaceObjects),
	}
	crdInfo, err := clusterconfig.NewCRDInfo(
		decode.NewGenericResourceDecoder(scheme.Scheme),
		currentConfigs.CRDClusterConfig,
		crds)
	p.errors = status.Append(p.errors, err)
	visitors = append(visitors, tree.NewCRDClusterConfigInfoVisitor(crdInfo))

	hierarchyConfigs := extractHierarchyConfigs(flatRoot.SystemObjects)
	visitors = append(visitors, p.opts.Extension.Visitors(hierarchyConfigs)...)

	visitors = append(visitors, transform.NewSyncGenerator())

	return visitors
}

// HydrateRootAndFlatten hydrates configuration into a fully-configured Root with the passed visitors.
func (p *Parser) HydrateRootAndFlatten(visitors []ast.Visitor, clusterName string) []ast.FileObject {
	astRoot := &ast.Root{
		ClusterName: clusterName,
	}

	root := p.runVisitors(astRoot, visitors)

	fileObjects := root.Flatten()

	crds, err := customresources.GetCRDs(fileObjects)
	if err != nil {
		// We couldn't read the CRDs, so we can't continue without showing a lot of
		// bogus errors to the user.
		p.errors = status.Append(p.errors, err)
		return nil
	}
	scoper := p.getScoper(crds...)
	fileObjects, selErr := resolveSelectors(scoper, clusterName, fileObjects)
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

func resolveSelectors(scoper utildiscovery.Scoper, clusterName string, fileObjects []ast.FileObject) ([]ast.FileObject, status.MultiError) {
	annErr := nonhierarchical.NewSelectorAnnotationValidator(scoper).Validate(fileObjects)
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

	return transform.RemoveEphemeralResources(fileObjects), nil
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// * system/ (flat, required)
// * cluster/ (flat, optional)
// * clusterregistry/ (flat, optional)
// * namespaces/ (recursive, optional)
func (p *Parser) Parse(
	importToken string,
	currentConfigs *namespaceconfig.AllConfigs,
	loadTime metav1.Time,
	clusterName string,
) (*namespaceconfig.AllConfigs, status.MultiError) {
	p.errors = nil

	flatRoot := p.ReadObjects()
	crds, err := customresources.GetCRDs(flatRoot.ClusterObjects)
	p.errors = status.Append(p.errors, err)
	if p.errors != nil {
		return nil, p.errors
	}

	visitors := p.GenerateVisitors(flatRoot, currentConfigs, crds)
	if p.errors != nil {
		return nil, p.errors
	}
	fileObjects := p.HydrateRootAndFlatten(visitors, clusterName)
	if p.errors != nil {
		return nil, p.errors
	}
	scoper := p.getScoper(crds...)
	configs, errs := namespaceconfig.NewAllConfigs(importToken, loadTime, scoper, fileObjects)
	if glog.V(8) {
		// REALLY useful when debugging.
		glog.Warningf("AllConfigs: %v", spew.Sdump(configs))
	}
	return configs, errs
}

func (p *Parser) getScoper(crds ...*v1beta1.CustomResourceDefinition) utildiscovery.Scoper {
	lists, discoveryErr := utildiscovery.GetResourcesFromClientGetter(p.clientGetter)
	if discoveryErr != nil {
		p.errors = status.Append(p.errors, discoveryErr)
		return nil
	}
	scoper, err := utildiscovery.NewScoperFromServerResources(lists, utildiscovery.ScopesFromCRDs(crds)...)
	if err != nil {
		p.errors = status.Append(p.errors, err)
		return nil
	}
	return scoper
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

func (p *Parser) readSystemResources() []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(repo.SystemDir)))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readNamespaceResources(crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(p.opts.Extension.NamespacesDir())))
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readClusterResources(crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(repo.ClusterDir)))
	p.errors = status.Append(p.errors, errs)
	return result
}

// ReadClusterRegistryResources reads the manifests declared in clusterregistry/.
func (p *Parser) ReadClusterRegistryResources() []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(repo.ClusterRegistryDir)))
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
