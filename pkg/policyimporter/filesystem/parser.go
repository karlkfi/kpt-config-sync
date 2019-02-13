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
	errors          *multierror.Builder
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
	p.errors = &multierror.Builder{}
	rootPath, err := nomospath.NewRoot(root)
	p.errors.Add(err)

	astRoot := &ast.Root{
		ImportToken: importToken,
		LoadTime:    loadTime,
		System:      &ast.System{},
		Cluster:     &ast.Cluster{},
	}
	// Always make sure we're getting the freshest data.
	p.discoveryClient.Invalidate()
	validateInstallation(p.discoveryClient, p.errors)
	if p.errors.Build() != nil {
		return nil, p.errors.Build()
	}

	hierarchyConfigs := extractHierarchyConfigs(p.readSystemResources(rootPath))
	p.errors.Add(addScope(astRoot, p.discoveryClient))

	visitors := []ast.Visitor{
		tree.NewSystemBuilderVisitor(p.readSystemResources(rootPath)),
		tree.NewClusterBuilderVisitor(p.readClusterResources(rootPath)),
		tree.NewClusterRegistryBuilderVisitor(p.readClusterRegistryResources(rootPath)),
		tree.NewBuilderVisitor(p.readNamespaceResources(rootPath)),
	}
	visitors = append(visitors, p.opts.Extension.Visitors(hierarchyConfigs, p.opts.Vet)...)
	outputVisitor := backend.NewOutputVisitor()
	visitors = append(visitors, outputVisitor)
	err = p.runVisitors(astRoot, visitors)
	if err != nil {
		return nil, err
	}

	policies := outputVisitor.AllPolicies()
	if glog.V(8) {
		// REALLY useful when debugging.
		glog.Warningf("allPolicies: %v", spew.Sdump(policies))
		glog.Warningf("all errors: %v", spew.Sdump(p.errors.Build()))
	}
	return policies, nil
}

func (p *Parser) runVisitors(root *ast.Root, visitors []ast.Visitor) error {
	for _, visitor := range visitors {
		if p.errors.HasErrors() && visitor.RequiresValidState() {
			return p.errors.Build()
		}
		root = root.Accept(visitor)
		p.errors.Add(visitor.Error())
		if visitor.Fatal() {
			return p.errors.Build()
		}
	}
	return nil
}

func (p *Parser) readSystemResources(root nomospath.Root) []ast.FileObject {
	return p.readResources(root.Join(repo.SystemDir))
}

func (p *Parser) readNamespaceResources(root nomospath.Root) []ast.FileObject {
	return p.readResources(root.Join(p.opts.Extension.NamespacesDir()))
}

func (p *Parser) readClusterResources(root nomospath.Root) []ast.FileObject {
	return p.readResources(root.Join(repo.ClusterDir))
}

func (p *Parser) readClusterRegistryResources(root nomospath.Root) []ast.FileObject {
	return p.readResources(root.Join(repo.ClusterRegistryDir))
}

// readResources walks dir recursively, looking for resources, and builds FileInfos from them.
func (p *Parser) readResources(dir nomospath.Relative) []ast.FileObject {
	// If there aren't any resources, skip builder, because builder treats that as an error.
	if _, err := os.Stat(dir.AbsoluteOSPath()); os.IsNotExist(err) {
		// Return empty list if unable to read directory
		return nil
	} else if err != nil {
		// If there was another error reading the directory, give up parsing the dir
		p.errors.Add(err)
		return nil
	}

	visitors, err := resource.ExpandPathsToFileVisitors(
		nil, dir.AbsoluteOSPath(), true, resource.FileExtensions, kubevalidation.NullSchema{})
	if err != nil {
		p.errors.Add(err)
		return nil
	}

	var fileObjects []ast.FileObject
	if len(visitors) > 0 {
		s, err := p.factory.Validator(p.opts.Validate)
		if err != nil {
			p.errors.Add(errors.Wrap(err, "failed to get schema"))
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
		p.errors.Add(err)
		for _, info := range fileInfos {
			// Assign relative path since that's what we actually need.
			source, err := dir.Root().Rel(info.Source)
			p.errors.Add(err)
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

// toInheritanceSpecs converts HierarchyConfigs to InheritanceSpecs. It also evaluates defaults so that later
// code doesn't have to.
func toInheritanceSpecs(configs []*v1alpha1.HierarchyConfig) map[schema.GroupKind]*transform.InheritanceSpec {
	specs := map[schema.GroupKind]*transform.InheritanceSpec{}
	for _, config := range configs {
		for _, r := range config.Spec.Resources {
			for _, k := range r.Kinds {
				gk := schema.GroupKind{Group: r.Group, Kind: k}
				var effectiveMode v1alpha1.HierarchyModeType
				if r.HierarchyMode == v1alpha1.HierarchyModeDefault {
					effectiveMode = v1alpha1.HierarchyModeInherit
				} else {
					effectiveMode = r.HierarchyMode
				}
				specs[gk] = &transform.InheritanceSpec{Mode: effectiveMode}
			}
		}
	}
	return specs
}

// validateInstallation checks to see if Nomos is installed properly.
// TODO(b/123598820): Server-side validation for this check.
func validateInstallation(resources discovery.ServerResourcesInterface, eb *multierror.Builder) {
	gv := v1alpha1.SchemeGroupVersion.String()
	_, err := resources.ServerResourcesForGroupVersion(gv)
	if err != nil {
		eb.Add(vet.PolicyManagementNotInstalledError{Err: errors.Wrapf(err, "unable to read required %s resources from cluster", gv)})
	}
}
