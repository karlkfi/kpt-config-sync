// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/backend"
	"github.com/google/nomos/pkg/importer/analyzer/transform"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
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
	errors       status.MultiError
}

// ParserOpt has often customizes the behavior of Parser.Parse.
type ParserOpt struct {
	// Vet turns on vetting mode, which catches a wider range of cross-cluster errors.
	Vet bool
	// Validate will raise validation errors if set.
	Validate bool
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
		opts:         opts,
	}
	return p
}

func (p *Parser) getBuilder(stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) *resource.Builder {
	return resource.NewBuilder(importer.NewFilesystemCRDAwareClientGetter(p.clientGetter, stubMissing, crds...))
}

// crtdsInRepo parses the cluster directory of the repo and returns all CustomResourceDefinitions it contains.
func (p *Parser) crdsInRepo() ([]*v1beta1.CustomResourceDefinition, status.Error) {
	fileObjects := p.readClusterResources(true)

	var crds []*v1beta1.CustomResourceDefinition
	for _, f := range fileObjects {
		object := f.Object
		if object.GetObjectKind().GroupVersionKind() != kinds.CustomResourceDefinition() {
			continue
		}

		crd, err := clusterconfig.AsCRD(object)
		if err != nil {
			return nil, status.PathWrapf(err, f.SlashPath())
		}
		crds = append(crds, crd)
	}
	return crds, nil
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// * system/ (flat, required)
// * cluster/ (flat, optional)
// * clusterregistry/ (flat, optional)
// * namespaces/ (recursive, optional)
func (p *Parser) Parse(importToken string, currentConfigs *namespaceconfig.AllConfigs,
	loadTime time.Time) (*namespaceconfig.AllConfigs, status.MultiError) {
	p.errors = nil

	// We need to retrieve the CRDs in the repo so we can also use them for resource discovery,
	// if we haven't yet added the CRDs to the cluster.
	crds, cErr := p.crdsInRepo()
	if cErr != nil {
		p.errors = status.Append(p.errors, cErr)
		return nil, p.errors
	}

	if p.errors != nil {
		return nil, p.errors
	}

	astRoot := &ast.Root{
		ImportToken: importToken,
		LoadTime:    loadTime,
		ClusterName: os.Getenv("CLUSTER_NAME"),
	}

	hierarchyConfigs := extractHierarchyConfigs(p.readSystemResources())
	crdInfo, err := clusterconfig.NewCRDInfo(
		decode.NewGenericResourceDecoder(scheme.Scheme),
		currentConfigs.CRDClusterConfig,
		crds)
	p.errors = status.Append(p.errors, err)

	discoveryClient, dErr := p.discoveryClient(crds...)
	p.errors = status.Append(p.errors, dErr)

	if p.errors != nil {
		return nil, p.errors
	}

	visitors := []ast.Visitor{
		tree.NewSystemBuilderVisitor(p.readSystemResources()),
		tree.NewClusterBuilderVisitor(p.readClusterResources(false, crds...)),
		tree.NewClusterRegistryBuilderVisitor(p.readClusterRegistryResources()),
		tree.NewBuilderVisitor(p.readNamespaceResources(crds...)),
		tree.NewAPIInfoBuilderVisitor(discoveryClient, transform.EphemeralResources()),
		tree.NewCRDClusterConfigInfoVisitor(crdInfo),
	}
	visitors = append(visitors, p.opts.Extension.Visitors(hierarchyConfigs, p.opts.Vet)...)

	outputVisitor := backend.NewOutputVisitor()
	visitors = append(visitors,
		transform.NewSyncGenerator(),
		outputVisitor)

	p.runVisitors(astRoot, visitors)
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
	return p.readResources(p.opts.RootPath.Join(cmpath.FromSlash(repo.SystemDir)), false)
}

func (p *Parser) readNamespaceResources(crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	return p.readResources(p.opts.RootPath.Join(cmpath.FromSlash(p.opts.Extension.NamespacesDir())), false, crds...)
}

func (p *Parser) readClusterResources(stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) []ast.
	FileObject {
	return p.readResources(p.opts.RootPath.Join(cmpath.FromSlash(repo.ClusterDir)), stubMissing, crds...)
}

func (p *Parser) readClusterRegistryResources() []ast.FileObject {
	return p.readResources(p.opts.RootPath.Join(cmpath.FromSlash(repo.ClusterRegistryDir)), false)
}

// readResources walks dir recursively, looking for resources, and builds FileInfos from them.
func (p *Parser) readResources(dir cmpath.Relative, stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	// If there aren't any resources, skip builder, because builder treats that as an error.
	if _, err := os.Stat(dir.AbsoluteOSPath()); os.IsNotExist(err) {
		// Return empty list if unable to read directory
		return nil
	} else if err != nil {
		// If there was another error reading the directory, give up parsing the dir
		p.errors = status.Append(p.errors, status.PathWrapf(err, dir.AbsoluteOSPath()))
		return nil
	}

	visitors, err := resource.ExpandPathsToFileVisitors(
		nil, dir.AbsoluteOSPath(), true, resource.FileExtensions, nil)
	if err != nil {
		p.errors = status.Append(p.errors, status.PathWrapf(err, dir.AbsoluteOSPath()))
		return nil
	}

	var fileObjects []ast.FileObject
	if len(visitors) > 0 {
		options := &resource.FilenameOptions{Recursive: true, Filenames: []string{dir.AbsoluteOSPath()}}
		crdBuilder := p.getBuilder(stubMissing, crds...)
		result := crdBuilder.
			Unstructured().
			ContinueOnError().
			FilenameParam(false, options).
			Do()
		fileInfos, err := result.Infos()
		p.errors = status.Append(p.errors, status.APIServerWrapf(err, "failed to get resource infos"))
		for _, info := range fileInfos {
			// Assign relative path since that's what we actually need.
			source, err := dir.Root().Rel(cmpath.FromOS(info.Source))
			p.errors = status.Append(p.errors, err)
			if err != nil {
				continue
			}
			object := asDefaultVersionedOrOriginal(info.Object, info.Mapping)
			fileObject := ast.NewFileObjectUnstructured(object, info.Object.(runtime.Unstructured), source.Path())
			fileObjects = append(fileObjects, fileObject)
		}
	}
	return fileObjects
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
func (p *Parser) ValidateInstallation() status.MultiError {
	resources, err := p.discoveryClient()
	if err != nil {
		return status.From(err)
	}

	gv := v1.SchemeGroupVersion.String()
	_, rErr := resources.ServerResourcesForGroupVersion(gv)
	if rErr != nil {
		if apierrors.IsNotFound(rErr) {
			return status.From(vet.ConfigManagementNotInstalledError(
				errors.Errorf("no resources exist on cluster with apiVersion: %s", gv)))
		}
		return status.From(vet.ConfigManagementNotInstalledError(rErr))
	}
	return nil
}

func (p *Parser) discoveryClient(crds ...*v1beta1.CustomResourceDefinition) (discovery.ServerResourcesInterface, error) {
	discoveryClient, dErr := importer.NewFilesystemCRDAwareClientGetter(p.clientGetter, true, crds...).ToDiscoveryClient()
	if dErr != nil {
		p.errors = status.Append(p.errors, status.APIServerWrapf(dErr, "could not get discovery client"))
		return nil, p.errors
	}

	// Always make sure we're getting the freshest data.
	discoveryClient.Invalidate()
	return discoveryClient, nil
}

// asDefaultVersionedOrOriginal returns the object as a Go object in the external form if possible (matching the
// group version kind of the mapping if provided, a best guess based on serialization if not provided, or obj if it cannot be converted.
func asDefaultVersionedOrOriginal(obj runtime.Object, mapping *meta.RESTMapping) runtime.Object {
	converter := runtime.ObjectConvertor(scheme.Scheme)
	groupVersioner := runtime.GroupVersioner(schema.GroupVersions(scheme.Scheme.PrioritizedVersionsAllGroups()))
	if mapping != nil {
		groupVersioner = mapping.GroupVersionKind.GroupVersion()
	}

	if cObj, err := converter.ConvertToVersion(obj, groupVersioner); err == nil {
		return cObj
	}
	return obj
}
