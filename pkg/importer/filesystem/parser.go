// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/backend"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
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
	clientGetter genericclioptions.RESTClientGetter
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
func NewParser(c genericclioptions.RESTClientGetter, opts ParserOpt) *Parser {
	p := &Parser{
		clientGetter: c,
		reader:       &FileReader{ClientGetter: c},
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
func (p *Parser) ReadObjects(crds []*v1beta1.CustomResourceDefinition) *ast.FlatRoot {
	return &ast.FlatRoot{
		SystemObjects:          p.readSystemResources(),
		ClusterRegistryObjects: p.ReadClusterRegistryResources(),
		ClusterObjects:         p.readClusterResources(crds...),
		NamespaceObjects:       p.readNamespaceResources(crds...),
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

	discoveryClient, dErr := p.discoveryClient(crds...)
	p.errors = status.Append(p.errors, dErr)
	visitors = append(visitors, tree.NewAPIInfoBuilderVisitor(discoveryClient, transform.EphemeralResources()))

	hierarchyConfigs := extractHierarchyConfigs(flatRoot.SystemObjects)
	visitors = append(visitors, p.opts.Extension.Visitors(hierarchyConfigs)...)

	visitors = append(visitors, transform.NewSyncGenerator())

	return visitors
}

// HydrateRoot hydrates configuration into a fully-configured Root with the passed visitors.
func (p *Parser) HydrateRoot(
	visitors []ast.Visitor,
	importToken string,
	loadTime time.Time,
	clusterName string,
) *ast.Root {
	astRoot := &ast.Root{
		ImportToken: importToken,
		LoadTime:    loadTime,
		ClusterName: clusterName,
	}

	p.runVisitors(astRoot, visitors)

	return astRoot
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
	loadTime time.Time,
	clusterName string,
) (*namespaceconfig.AllConfigs, status.MultiError) {
	p.errors = nil

	// We need to retrieve the CRDs in the repo so we can also use them for resource discovery,
	// if we haven't yet added the CRDs to the cluster.
	crds, cErr := readCRDs(p.reader, p.opts.RootPath.Join(cmpath.FromSlash(repo.ClusterDir)))
	if cErr != nil {
		p.errors = status.Append(p.errors, cErr)
		return nil, p.errors
	}
	if p.errors != nil {
		return nil, p.errors
	}

	flatRoot := p.ReadObjects(crds)
	if p.errors != nil {
		return nil, p.errors
	}

	visitors := p.GenerateVisitors(flatRoot, currentConfigs, crds)
	outputVisitor := backend.NewOutputVisitor()
	visitors = append(visitors, outputVisitor)

	p.HydrateRoot(visitors, importToken, loadTime, clusterName)
	if p.errors != nil {
		return nil, p.errors
	}

	configs := outputVisitor.AllConfigs()
	if glog.V(8) {
		// REALLY useful when debugging.
		glog.Warningf("AllConfigs: %v", spew.Sdump(configs))
	}
	return configs, nil
}

func (p *Parser) runVisitors(root *ast.Root, visitors []ast.Visitor) {
	for _, visitor := range visitors {
		if p.errors != nil && visitor.RequiresValidState() {
			return
		}
		root = root.Accept(visitor)
		p.errors = status.Append(p.errors, visitor.Error())
		if visitor.Fatal() {
			return
		}
	}
}

func (p *Parser) readSystemResources() []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(repo.SystemDir)), false)
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readNamespaceResources(crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(p.opts.Extension.NamespacesDir())), false, crds...)
	p.errors = status.Append(p.errors, errs)
	return result
}

func (p *Parser) readClusterResources(crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(repo.ClusterDir)), false, crds...)
	p.errors = status.Append(p.errors, errs)
	return result
}

// ReadClusterRegistryResources reads the manifests declared in clusterregistry/.
func (p *Parser) ReadClusterRegistryResources() []ast.FileObject {
	result, errs := p.reader.Read(p.opts.RootPath.Join(cmpath.FromSlash(repo.ClusterRegistryDir)), false)
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

// ValidateInstallation checks to see if Nomos is installed properly.
// TODO(b/123598820): Server-side validation for this check.
func ValidateInstallation(client genericclioptions.RESTClientGetter) status.MultiError {
	discoveryClient, err := client.ToDiscoveryClient()
	if err != nil {
		return status.APIServerError(err, "could not get discovery client")
	}

	gv := v1.SchemeGroupVersion.String()
	_, rErr := discoveryClient.ServerResourcesForGroupVersion(gv)
	if rErr != nil {
		if apierrors.IsNotFound(rErr) {
			return ConfigManagementNotInstalledError(
				errors.Errorf("no resources exist on cluster with apiVersion: %s", gv))
		}
		return ConfigManagementNotInstalledError(rErr)
	}
	return nil
}

func (p *Parser) discoveryClient(crds ...*v1beta1.CustomResourceDefinition) (discovery.ServerResourcesInterface, error) {
	discoveryClient, dErr := importer.NewFilesystemCRDAwareClientGetter(p.clientGetter, true, crds...).ToDiscoveryClient()
	if dErr != nil {
		p.errors = status.Append(p.errors, status.APIServerError(dErr, "could not get discovery client"))
		return nil, p.errors
	}

	// Always make sure we're getting the freshest data.
	discoveryClient.Invalidate()
	return discoveryClient, nil
}
