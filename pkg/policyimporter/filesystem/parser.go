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
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/backend"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/coverage"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
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
	// root is the path to Nomos root.
	root nomospath.Root
}

// ParserOpt has often customized parser options. Use for example in NewParser.
type ParserOpt struct {
	// Vet turns on vetting mode, which catches a wider range of cross-cluster errors.
	Vet bool
	// Validate will raise validation errors if set.
	Validate bool
	// Extension is the ParserConfig object that the parser will consume for configuring various
	// aspects of the execution (see ParserConfig).
	Extension ParserConfig
}

// NewParser creates a new Parser.
// clientConfig can be used to configure api server client. It should be set to nil when running in cluster.
// resources is the list returned by the DisoveryClient ServerResources call which represents resources
// 		that are returned by the API server during discovery.
// opts turns on options for the parser.
func NewParser(clientGetter genericclioptions.RESTClientGetter, opts ParserOpt) (*Parser, error) {
	return NewParserWithFactory(cmdutil.NewFactory(clientGetter), opts)
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

func toDirInfoMap(fileInfos []ast.FileObject) map[nomospath.Relative][]ast.FileObject {
	result := make(map[nomospath.Relative][]ast.FileObject)

	// If a directory has resources, its value in the map will be non-nil.
	for _, i := range fileInfos {
		d := i.Dir()
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
func (p *Parser) Parse(root string, importToken string, loadTime time.Time) (*v1.AllPolicies, error) {
	r, err := nomospath.NewRoot(root)
	if err != nil {
		return nil, errors.Wrap(err, "unable to use as Nomos root")
	}
	p.root = r

	astRoot := &ast.Root{
		Cluster:     &ast.Cluster{},
		ImportToken: importToken,
		LoadTime:    loadTime,
	}
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
	systemInfos := p.readRequiredResources(p.root.Join(repo.SystemDir), &errorBuilder)
	astRoot.System = getSystemDir(systemInfos)
	astRoot.Repo, syncs = processSystem(systemInfos, p.opts, apiInfo, &errorBuilder)
	if errorBuilder.HasErrors() {
		// Don't continue processing if any errors encountered processing system/
		return nil, errorBuilder.Build()
	}

	// processing for <root>/cluster/*
	clusterDir := p.root.Join(repo.ClusterDir)
	clusterInfos := p.readResources(clusterDir, &errorBuilder)
	validateCluster(clusterInfos, &errorBuilder)

	// processing for <root>/clusterregistry/*
	clusterregistryInfos := p.readResources(p.root.Join(repo.ClusterRegistryDir), &errorBuilder)
	validateClusterRegistry(clusterregistryInfos, &errorBuilder)
	clusters := getClusters(clusterregistryInfos)
	selectors := getSelectors(clusterregistryInfos)
	astRoot.ClusterRegistry = getClusterRegistry(clusterregistryInfos)
	cs, err := sel.NewClusterSelectors(clusters, selectors, os.Getenv("CLUSTER_NAME"))
	// TODO(b/120229144): To be factored into KNV Error.
	errorBuilder.Add(errors.Wrapf(err, "could not create cluster selectors"))
	sel.SetClusterSelector(cs, astRoot)

	nsDir := p.root.Join(p.opts.Extension.NamespacesDir())
	nsDirsOrdered := p.allDirs(nsDir, &errorBuilder)

	nsInfos := p.readResources(nsDir, &errorBuilder)
	validateNamespaces(nsInfos, nsDirsOrdered,
		coverage.NewForCluster(clusters, selectors, &errorBuilder), &errorBuilder)

	// TODO: temporary until processDirs refactoring
	dirInfos := toDirInfoMap(nsInfos)

	policies, err := p.processDirs(apiInfo, dirInfos, clusterInfos, nsDirsOrdered, astRoot, syncs, clusters, selectors)
	errorBuilder.Add(err)

	if glog.V(8) {
		// REALLY useful when debugging.
		glog.Warningf("allPolicies: %v", spew.Sdump(policies))
		glog.Warningf("all errors: %v", spew.Sdump(errorBuilder.Build()))
	}
	if errorBuilder.HasErrors() {
		return nil, errorBuilder.Build()
	}
	return policies, nil
}

// readRequiredResources walks dir recursively, looking for resources, and builds FileInfos from them.
// Returns an error if the directory is missing.
func (p *Parser) readRequiredResources(dir nomospath.Relative, errorBuilder *multierror.Builder) []ast.FileObject {
	if _, err := os.Stat(dir.AbsoluteOSPath()); os.IsNotExist(err) {
		errorBuilder.Add(vet.MissingDirectoryError{})
		return nil
	}
	return p.readResources(dir, errorBuilder)
}

// readResources walks dir recursively, looking for resources, and builds FileInfos from them.
func (p *Parser) readResources(dir nomospath.Relative, errorBuilder *multierror.Builder) []ast.FileObject {
	// If there aren't any resources, skip builder, because builder treats that as an error.
	if _, err := os.Stat(dir.AbsoluteOSPath()); os.IsNotExist(err) {
		// Return empty list if unable to read directory
		return nil
	} else if err != nil {
		// If there was another error reading the directory, give up parsing the dir
		errorBuilder.Add(err)
		return nil
	}

	visitors, err := resource.ExpandPathsToFileVisitors(
		nil, dir.AbsoluteOSPath(), true, resource.FileExtensions, kubevalidation.NullSchema{})
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
		options := &resource.FilenameOptions{Recursive: true, Filenames: []string{dir.AbsoluteOSPath()}}
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
			source, err := p.root.Rel(info.Source)
			errorBuilder.Add(err)
			if err != nil {
				continue
			}
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
	dirInfos map[nomospath.Relative][]ast.FileObject,
	clusterObjects []ast.FileObject,
	nsDirsOrdered []nomospath.Relative,
	astRoot *ast.Root,
	syncs []*v1alpha1.Sync,
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector) (*v1.AllPolicies, error) {

	processCluster(clusterObjects, astRoot)

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

	tree := treeGenerator.Build(&errorBuilder)
	astRoot.Tree = tree

	visitors := p.opts.Extension.Visitors(syncs, clusters, selectors, p.opts.Vet, apiInfo)
	for _, visitor := range visitors {
		astRoot = astRoot.Accept(visitor)
		if err := visitor.Error(); err != nil {
			errorBuilder.Add(err)
			return nil, errorBuilder.Build()
		}
	}

	outputVisitor := backend.NewOutputVisitor(syncs)
	astRoot.Accept(outputVisitor)
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
					effectiveMode = v1alpha1.HierarchyModeInherit
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
func (p *Parser) allDirs(nsDir nomospath.Relative, errorBuilder *multierror.Builder) []nomospath.Relative {
	var paths []nomospath.Relative
	err := filepath.Walk(nsDir.AbsoluteOSPath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			rel, err := p.root.Rel(path)
			if err != nil {
				return err
			}
			paths = append(paths, rel)
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		errorBuilder.Add(err)
		return nil
	}
	return paths
}
