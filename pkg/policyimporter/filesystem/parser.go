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
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	utildiscovery "github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/util/multierror"
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
	errorBuilder := &multierror.Builder{}

	// Always make sure we're getting the freshest data.
	p.discoveryClient.Invalidate()
	errorBuilder.Add(addScope(astRoot, p.discoveryClient))

	// processing for <root>/system/*
	systemInfos := p.readSystemResources(errorBuilder)
	astRoot.Accept(tree.NewSystemBuilderVisitor(systemInfos))

	// TODO: Delete these lines once syncs are defunct.
	validateSyncs(astRoot, systemInfos, errorBuilder)
	syncs := processSyncs(astRoot, systemInfos, p.opts)

	// processing for <root>/cluster/*
	clusterInfos := p.readClusterResources(errorBuilder)
	astRoot.Accept(tree.NewClusterBuilderVisitor(clusterInfos))

	// processing for <root>/clusterregistry/*
	clusterregistryInfos := p.readClusterRegistryResources(errorBuilder)

	astRoot.Accept(tree.NewClusterRegistryBuilderVisitor(clusterregistryInfos))
	selectorAdder := sel.NewClusterSelectorAdder()
	astRoot.Accept(selectorAdder)
	errorBuilder.Add(selectorAdder.Error())

	nsInfos := p.readNamespaceResources(errorBuilder)

	visitors := []ast.Visitor{tree.NewBuilderVisitor(nsInfos)}
	visitors = append(visitors, p.opts.Extension.Visitors(syncs, p.opts.Vet)...)
	for _, visitor := range visitors {
		if errorBuilder.HasErrors() && visitor.RequiresValidState() {
			return nil, errorBuilder.Build()
		}
		astRoot = astRoot.Accept(visitor)
		errorBuilder.Add(visitor.Error())
		if visitor.Fatal() {
			return nil, errorBuilder.Build()
		}
	}

	outputVisitor := backend.NewOutputVisitor()
	astRoot.Accept(outputVisitor)
	policies := outputVisitor.AllPolicies()

	if errorBuilder.HasErrors() {
		return nil, errorBuilder.Build()
	}

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

func (p *Parser) readSystemResources(eb *multierror.Builder) []ast.FileObject {
	return p.readResources(p.root.Join(repo.SystemDir), eb)
}

func (p *Parser) readNamespaceResources(eb *multierror.Builder) []ast.FileObject {
	return p.readResources(p.root.Join(p.opts.Extension.NamespacesDir()), eb)
}

func (p *Parser) readClusterResources(eb *multierror.Builder) []ast.FileObject {
	return p.readResources(p.root.Join(repo.ClusterDir), eb)
}

func (p *Parser) readClusterRegistryResources(eb *multierror.Builder) []ast.FileObject {
	return p.readResources(p.root.Join(repo.ClusterRegistryDir), eb)
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

func addScope(root *ast.Root, client discovery.ServerResourcesInterface) error {
	resources, discoveryErr := client.ServerResources()
	if discoveryErr != nil {
		return vet.UndocumentedWrapf(discoveryErr, "failed to get server resources")
	}

	resources = append(resources, transform.EphemeralResources()...)
	apiInfo, err := utildiscovery.NewAPIInfo(resources)
	if err != nil {
		return err
	}
	utildiscovery.AddAPIInfo(root, apiInfo)
	return nil
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
