/*
Copyright 2017 The CSP Config Management Authors.
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
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
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
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	kubevalidation "k8s.io/kubernetes/pkg/kubectl/validation"
)

func init() {
	// Add Nomos types to the Scheme used by asDefaultVersionedOrOriginal for
	// converting Unstructured to specific types.
	utilruntime.Must(v1beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(v1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterregistry.AddToScheme(scheme.Scheme))
}

type factoryFactory func(...*v1beta1.CustomResourceDefinition) cmdutil.Factory

// Parser reads files on disk and builds Nomos Config objects to be reconciled by the Syncer.
type Parser struct {
	opts           ParserOpt
	factoryFactory factoryFactory
	errors         status.MultiError
}

// ParserOpt has often customized parser options. Use for example in NewParser.
type ParserOpt struct {
	// Vet turns on vetting mode, which catches a wider range of cross-cluster errors.
	Vet bool
	// Validate will raise validation errors if set.
	Validate bool
	// Extension is the ParserConfig object that the parser will consume for configuring various
	// aspects of the execution (see ParserConfig).
	Extension  ParserConfig
	EnableCRDs bool
}

// NewParser creates a new Parser using the specified factoryFactory and parser options.
func NewParser(f factoryFactory, opts ParserOpt) (*Parser, error) {
	p := &Parser{
		factoryFactory: f,
		opts:           opts,
	}
	return p, nil
}

// crtdsInRepo parses the cluster directory of the repo and returns all CustomResourceDefinitions it contains.
func (p *Parser) crdsInRepo(rootPath cmpath.Root) ([]*v1beta1.CustomResourceDefinition, status.Error) {
	fileObjects := p.readClusterResources(rootPath)

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
func (p *Parser) Parse(root string, importToken string, currentConfigs *namespaceconfig.AllConfigs,
	loadTime time.Time) (*namespaceconfig.AllConfigs, status.MultiError) {
	p.errors = nil
	rootPath, err := cmpath.NewRoot(cmpath.FromOS(root))
	p.errors = status.Append(p.errors, err)

	// We need to retrieve the CRDs in the repo so we can also use them for resource discovery,
	// if we haven't yet added the CRDs to the cluster.
	crds, cErr := p.crdsInRepo(rootPath)
	if cErr != nil {
		p.errors = status.Append(p.errors, cErr)
		return nil, p.errors
	}

	discoveryClient, dErr := p.factoryFactory().ToDiscoveryClient()
	if dErr != nil {
		p.errors = status.Append(p.errors, status.APIServerWrapf(dErr, "could not get discovery client"))
		return nil, p.errors
	}

	// Always make sure we're getting the freshest data.
	discoveryClient.Invalidate()
	p.errors = status.Append(p.errors, validateInstallation(discoveryClient))

	if p.errors != nil {
		return nil, p.errors
	}

	astRoot := &ast.Root{
		ImportToken: importToken,
		LoadTime:    loadTime,
	}
	astRoot.Data, err = ast.Add(astRoot.Data, RootPath{}, rootPath)
	p.errors = status.Append(p.errors, err)

	hierarchyConfigs := extractHierarchyConfigs(p.readSystemResources(rootPath))
	crdInfo, err := clusterconfig.NewCRDInfo(
		decode.NewGenericResourceDecoder(scheme.Scheme),
		currentConfigs.CRDClusterConfig,
		crds)
	p.errors = status.Append(p.errors, err)

	if p.errors != nil {
		return nil, p.errors
	}

	visitors := []ast.Visitor{
		tree.NewSystemBuilderVisitor(p.readSystemResources(rootPath)),
		tree.NewClusterBuilderVisitor(p.readClusterResources(rootPath, crds...)),
		tree.NewClusterRegistryBuilderVisitor(p.readClusterRegistryResources(rootPath)),
		tree.NewBuilderVisitor(p.readNamespaceResources(rootPath)),
		tree.NewAPIInfoBuilderVisitor(discoveryClient, transform.EphemeralResources(), p.opts.EnableCRDs),
		tree.NewCRDClusterConfigInfoVisitor(crdInfo),
	}
	visitors = append(visitors, p.opts.Extension.Visitors(hierarchyConfigs, p.opts.Vet, p.opts.EnableCRDs)...)

	outputVisitor := backend.NewOutputVisitor(p.opts.EnableCRDs)
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

func (p *Parser) readSystemResources(root cmpath.Root) []ast.FileObject {
	return p.readResources(root.Join(cmpath.FromSlash(repo.SystemDir)))
}

func (p *Parser) readNamespaceResources(root cmpath.Root, crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	return p.readResources(root.Join(cmpath.FromSlash(p.opts.Extension.NamespacesDir())), crds...)
}

func (p *Parser) readClusterResources(root cmpath.Root, crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
	return p.readResources(root.Join(cmpath.FromSlash(repo.ClusterDir)), crds...)
}

func (p *Parser) readClusterRegistryResources(root cmpath.Root) []ast.FileObject {
	return p.readResources(root.Join(cmpath.FromSlash(repo.ClusterRegistryDir)))
}

// readResources walks dir recursively, looking for resources, and builds FileInfos from them.
func (p *Parser) readResources(dir cmpath.Relative, crds ...*v1beta1.CustomResourceDefinition) []ast.FileObject {
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
		nil, dir.AbsoluteOSPath(), true, resource.FileExtensions, kubevalidation.NullSchema{})
	if err != nil {
		p.errors = status.Append(p.errors, status.PathWrapf(err, dir.AbsoluteOSPath()))
		return nil
	}

	var fileObjects []ast.FileObject
	if len(visitors) > 0 {
		s, err := p.factoryFactory().Validator(p.opts.Validate)
		if err != nil {
			p.errors = status.Append(p.errors, status.APIServerWrapf(err, "failed to get schema"))
			return nil
		}
		options := &resource.FilenameOptions{Recursive: true, Filenames: []string{dir.AbsoluteOSPath()}}
		crdFactory := p.factoryFactory(crds...)
		result := crdFactory.NewBuilder().
			Unstructured().
			Schema(s).
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
			fileObject := ast.NewFileObject(object, source.Path())
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

// validateInstallation checks to see if Nomos is installed properly.
// TODO(b/123598820): Server-side validation for this check.
func validateInstallation(resources discovery.ServerResourcesInterface) status.MultiError {
	gv := v1.SchemeGroupVersion.String()
	_, err := resources.ServerResourcesForGroupVersion(gv)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return status.From(vet.ConfigManagementNotInstalledError{
				Err: errors.Errorf("no resources exist on cluster with apiVersion: %s", gv),
			})
		}
		return status.From(vet.ConfigManagementNotInstalledError{Err: err})
	}
	return nil
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
