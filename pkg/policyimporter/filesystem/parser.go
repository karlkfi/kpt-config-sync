/*
Copyright 2017 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	"os"
	"path/filepath"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/backend"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/meta"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/multierror"
	policynodevalidator "github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions/resource"
	kubevalidation "k8s.io/kubernetes/pkg/kubectl/validation"
)

func init() {
	// Add Nomos types to the Scheme used by util.AsDefaultVersionedOrOriginal for
	// converting Unstructured to specific types.
	utilruntime.Must(v1.AddToScheme(legacyscheme.Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(legacyscheme.Scheme))
	utilruntime.Must(clusterregistry.AddToScheme(legacyscheme.Scheme))
}

// Parser reads files on disk and builds Nomos CRDs.
type Parser struct {
	opts            ParserOpt
	factory         cmdutil.Factory
	discoveryClient discovery.CachedDiscoveryInterface
	// OS-specific path to root.
	root string
}

// ParserExtension extends the functionality of the parser by allowing the override of visitors or addition
// of sync resources.
// TODO(willbeason): Bespin requires the visitors to be overridden to avoid validators that
// cause a Bespin import to fail, but the resources need to be appended to. This
// isn't great. Ideally the visitors should be able to be chained as well, then
// the ParserOpt could take multiple ParserExtensions and run them all.
type ParserExtension interface {
	// Visitors *overrides* the normal visitor functionality of the parser.
	Visitors() VisitorProvider
	// SyncResources *appends* sync resources to the normal Nomos sync resources.
	SyncResources() []*v1alpha1.Sync
}

// ParserOpt has often customized parser options. Use for example in NewParser.
type ParserOpt struct {
	// Vet turns on vetting mode, which catches a wider range of cross-cluster errors.
	Vet bool
	// Validate will raise validation errors if set.
	Validate  bool
	Extension ParserExtension
}

// VisitorProvider is an interface that abstracts out the source of visitors
// that are used to walk the AST.
type VisitorProvider interface {
	visitors(apiInfo *meta.APIInfo) []ast.Visitor
}

// nomosVisitorProvider is the default visitor provider.  It handles
// plain vanilla nomos configs.
type nomosVisitorProvider struct {
	syncs     []*v1alpha1.Sync
	clusters  []clusterregistry.Cluster
	selectors []v1alpha1.ClusterSelector
	opts      ParserOpt
}

func (n nomosVisitorProvider) visitors(apiInfo *meta.APIInfo) []ast.Visitor {
	specs := toInheritanceSpecs(n.syncs)
	visitors := []ast.Visitor{
		validation.NewInputValidator(n.syncs, specs, n.clusters, n.selectors, n.opts.Vet),
		transform.NewPathAnnotationVisitor(),
		validation.NewScope(apiInfo),
		transform.NewClusterSelectorVisitor(), // Filter out unneeded parts of the tree
		transform.NewAnnotationInlinerVisitor(),
		transform.NewInheritanceVisitor(specs),
	}
	if spec, found := specs[kinds.ResourceQuota().GroupKind()]; found && spec.Mode == v1alpha1.HierarchyModeHierarchicalQuota {
		visitors = append(visitors, transform.NewQuotaVisitor())
	}
	visitors = append(visitors, validation.NewNameValidator())

	return visitors
}

// NewParser creates a new Parser.
// clientConfig can be used to configure api server client. It should be set to nil when running in cluster.
// resources is the list returned by the DisoveryClient ServerResources call which represents resources
// 		that are returned by the API server during discovery.
// opts turns on options for the parser.
func NewParser(clientGetter genericclioptions.RESTClientGetter, opts ParserOpt) (*Parser, error) {
	return NewParserWithFactory(cmdutil.NewFactory(clientGetter), opts)
}

func (p *Parser) nomosVisitorProvider(
	syncs []*v1alpha1.Sync,
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector) VisitorProvider {
	return nomosVisitorProvider{
		syncs:     syncs,
		clusters:  clusters,
		selectors: selectors,
		opts:      p.opts,
	}
}

// NewParserWithFactory creates a new Parser using the specified factory.
// NewParser is the more common constructor, but this is useful for testing.
func NewParserWithFactory(f cmdutil.Factory, opts ParserOpt) (*Parser, error) {
	dc, err := f.ToDiscoveryClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not get discovery client")
	}
	p := &Parser{
		opts:            opts,
		factory:         f,
		discoveryClient: dc,
	}
	return p, nil
}

func toDirInfoMap(fileInfos []ast.FileObject) map[string][]ast.FileObject {
	result := make(map[string][]ast.FileObject)

	// If a directory has resources, its value in the map will be non-nil.
	for _, i := range fileInfos {
		d := filepath.Dir(i.Source())
		result[d] = append(result[d], i)
	}

	return result
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// * system/ (flat, required)
// * cluster/ (flat, optional)
// * clusterregistry/ (flat, optional)
// * namespaces/ (recursive, optional)
func (p *Parser) Parse(root string) (*v1.AllPolicies, error) {
	p.root = root
	fsCtx := &ast.Root{Cluster: &ast.Cluster{}}
	errorBuilder := multierror.Builder{}

	// Always make sure we're getting the freshest data.
	p.discoveryClient.Invalidate()
	resources, discoveryErr := p.discoveryClient.ServerResources()
	if discoveryErr != nil {
		return nil, errors.Wrap(discoveryErr, "failed to get server resources")
	}
	apiInfo, err := meta.NewAPIInfo(resources)
	if err != nil {
		return nil, err
	}

	// processing for <root>/system/*
	var syncs []*v1alpha1.Sync
	systemInfos := p.readRequiredResources(filepath.Join(root, repo.SystemDir), &errorBuilder)
	fsCtx.Repo, syncs = processSystem(systemInfos, p.opts, apiInfo, &errorBuilder)
	if errorBuilder.HasErrors() {
		// Don't continue processing if any errors encountered processing system/
		return nil, errorBuilder.Build()
	}

	// processing for <root>/cluster/*
	clusterDir := filepath.Join(root, repo.ClusterDir)
	clusterInfos := p.readResources(clusterDir, &errorBuilder)
	validateCluster(clusterInfos, &errorBuilder)

	// processing for <root>/clusterregistry/*
	clusterregistryInfos := p.readResources(filepath.Join(root, repo.ClusterRegistryDir), &errorBuilder)
	validateClusterRegistry(clusterregistryInfos, &errorBuilder)
	clusters := getClusters(clusterregistryInfos)
	selectors := getSelectors(clusterregistryInfos)
	cs, err := sel.NewClusterSelectors(clusters, selectors, os.Getenv("CLUSTER_NAME"))
	// TODO(b/120229144): To be factored into KNV Error.
	errorBuilder.Add(errors.Wrapf(err, "could not create cluster selectors"))
	sel.SetClusterSelector(cs, fsCtx)

	nsDir := filepath.Join(root, repo.NamespacesDir)
	nsDirsOrdered := p.allDirs(nsDir, &errorBuilder)

	nsInfos := p.readResources(nsDir, &errorBuilder)
	validateNamespaces(nsInfos, nsDirsOrdered, &errorBuilder)

	// TODO: temporary until processDirs refactoring
	dirInfos := toDirInfoMap(nsInfos)
	vp := p.nomosVisitorProvider(syncs, clusters, selectors)
	if p.opts.Extension != nil && p.opts.Extension.Visitors() != nil {
		vp = p.opts.Extension.Visitors()
	}

	policies, err := p.processDirs(apiInfo, dirInfos, clusterInfos, vp, nsDirsOrdered, clusterDir, fsCtx, syncs)
	errorBuilder.Add(err)

	if errorBuilder.HasErrors() {
		return nil, errorBuilder.Build()
	}
	return policies, nil
}

func (p *Parser) relativePath(source string) string {
	r, err := filepath.Rel(p.root, source)
	if err != nil {
		panic(errors.Wrap(err, "programmer error"))
	}
	return r
}

// readRequiredResources walks dir recursively, looking for resources, and builds FileInfos from them.
// Returns an error if the directory is missing.
func (p *Parser) readRequiredResources(dir string, errorBuilder *multierror.Builder) []ast.FileObject {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		errorBuilder.Add(veterrors.MissingDirectoryError{})
		return nil
	}
	return p.readResources(dir, errorBuilder)
}

// readResources walks dir recursively, looking for resources, and builds FileInfos from them.
func (p *Parser) readResources(dir string, errorBuilder *multierror.Builder) []ast.FileObject {
	// If there aren't any resources, skip builder, because builder treats that as an error.
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Return empty list if unable to read directory
		return nil
	} else if err != nil {
		// If there was another error reading the directory, give up parsing the dir
		errorBuilder.Add(err)
		return nil
	}

	visitors, err := resource.ExpandPathsToFileVisitors(
		nil, dir, true, resource.FileExtensions, kubevalidation.NullSchema{})
	if err != nil {
		errorBuilder.Add(err)
		return nil
	}

	var fileObjects []ast.FileObject
	if len(visitors) > 0 {
		s, err := p.factory.Validator(p.opts.Validate)
		if err != nil {
			errorBuilder.Add(errors.Wrap(err, "failed to get schema"))
			return nil
		}
		options := &resource.FilenameOptions{Recursive: true, Filenames: []string{dir}}
		result := p.factory.NewBuilder().
			Unstructured().
			Schema(s).
			ContinueOnError().
			FilenameParam(false, options).
			Do()
		fileInfos, err := result.Infos()
		errorBuilder.Add(err)
		for _, info := range fileInfos {
			// Assign relative path since that's what we actually need.
			source := p.relativePath(info.Source)
			object := cmdutil.AsDefaultVersionedOrOriginal(info.Object, info.Mapping)
			fileObject := ast.NewFileObject(object, source)
			fileObjects = append(fileObjects, fileObject)
		}
	}
	return fileObjects
}

// processDirs validates objects in directory trees and converts them into hierarchical policy objects.
//
// clusterregistryInfos is the set of resources found in the directory <root>/clusterregistry.
//
// It first processes the cluster directory and then the tree hierarchy.
// cluster is a single, flat directory containing cluster-scoped resources.
// tree is hierarchical, containing 2 categories of directories:
// 1. AbstractNamespace directory: Non-leaf directories at any depth within root directory.
// 2. Namespace directory: Leaf directories at any depth within root directory.
func (p *Parser) processDirs(apiInfo *meta.APIInfo,
	dirInfos map[string][]ast.FileObject,
	clusterObjects []ast.FileObject,
	vp VisitorProvider,
	nsDirsOrdered []string,
	clusterDir string,
	fsRoot *ast.Root,
	syncs []*v1alpha1.Sync) (*v1.AllPolicies, error) {

	processCluster(clusterObjects, fsRoot)

	errorBuilder := multierror.Builder{}
	treeGenerator := NewDirectoryTree()
	if len(nsDirsOrdered) > 0 {
		rootDir := nsDirsOrdered[0]
		infos := dirInfos[rootDir]
		processNamespaces(rootDir, infos, treeGenerator, &errorBuilder)
		if errorBuilder.HasErrors() {
			return nil, errorBuilder.Build()
		}
		for _, d := range nsDirsOrdered[1:] {
			infos := dirInfos[d]
			processNamespaces(d, infos, treeGenerator, &errorBuilder)
			if errorBuilder.HasErrors() {
				return nil, errorBuilder.Build()
			}
		}
	}

	tree, err := treeGenerator.Build()
	if err != nil {
		errorBuilder.Add(errors.Wrapf(err, "failed to treeify policy nodes"))
		return nil, errorBuilder.Build()
	}
	fsRoot.Tree = tree

	visitors := vp.visitors(apiInfo)
	for _, visitor := range visitors {
		fsRoot = fsRoot.Accept(visitor)
		if err := visitor.Error(); err != nil {
			errorBuilder.Add(err)
			return nil, errorBuilder.Build()
		}
	}

	outputVisitor := backend.NewOutputVisitor(syncs)
	fsRoot.Accept(outputVisitor)
	policies := outputVisitor.AllPolicies()

	if err := clusterpolicy.Validate(policies.ClusterPolicy); err != nil {
		errorBuilder.Add(err)
		return nil, errorBuilder.Build()
	}
	v := policynodevalidator.FromMap(policies.PolicyNodes)
	if err := v.Validate(); err != nil {
		errorBuilder.Add(err)
		return nil, errorBuilder.Build()
	}

	if errorBuilder.HasErrors() {
		return nil, errorBuilder.Build()
	}
	return policies, nil
}

// toInheritanceSpecs converts Syncs to InheritanceSpecs. It also evaluates defaults so that later
// code doesn't have to.
func toInheritanceSpecs(syncs []*v1alpha1.Sync) map[schema.GroupKind]*transform.InheritanceSpec {
	specs := map[schema.GroupKind]*transform.InheritanceSpec{}
	for _, sync := range syncs {
		for _, sg := range sync.Spec.Groups {
			for _, k := range sg.Kinds {
				var effectiveMode v1alpha1.HierarchyModeType
				gk := schema.GroupKind{Group: sg.Group, Kind: k.Kind}
				if k.HierarchyMode == v1alpha1.HierarchyModeDefault {
					if gk == kinds.RoleBinding().GroupKind() {
						effectiveMode = v1alpha1.HierarchyModeInherit
					} else if gk == kinds.ResourceQuota().GroupKind() {
						effectiveMode = v1alpha1.HierarchyModeHierarchicalQuota
					} else {
						effectiveMode = v1alpha1.HierarchyModeNone
					}
				} else {
					effectiveMode = k.HierarchyMode
				}
				specs[gk] = &transform.InheritanceSpec{Mode: effectiveMode}
			}
		}
	}
	return specs
}

// allDirs returns absolute paths of all directories in root, in lexicographic (depth-first) order.
func (p *Parser) allDirs(nsDir string, errorBuilder *multierror.Builder) []string {
	var paths []string
	err := filepath.Walk(nsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			paths = append(paths, p.relativePath(path))
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		errorBuilder.Add(err)
		return nil
	}
	return paths
}
