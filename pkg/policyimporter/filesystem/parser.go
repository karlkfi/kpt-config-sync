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

	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/backend"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/meta"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/multierror"
	policynodevalidator "github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// unsupportedSyncResources is a set of GroupVersionKinds that are not allowed in Syncs.
var unsupportedSyncResources = map[schema.GroupVersionKind]bool{
	extensionsv1beta1.SchemeGroupVersion.WithKind("CustomResourceDefinition"): true,
	corev1.SchemeGroupVersion.WithKind("Namespace"):                           true,
}

func init() {
	// Add Nomos and Bespin types to the Scheme used by util.AsDefaultVersionedOrOriginal for
	// converting Unstructured to specific types.
	utilruntime.Must(v1.AddToScheme(legacyscheme.Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(legacyscheme.Scheme))
	utilruntime.Must(bespinv1.AddToScheme(legacyscheme.Scheme))
	utilruntime.Must(clusterregistry.AddToScheme(legacyscheme.Scheme))
}

// Parser reads files on disk and builds Nomos CRDs.
type Parser struct {
	opts            ParserOpt
	factory         cmdutil.Factory
	discoveryClient discovery.CachedDiscoveryInterface
	// OS-specific path to root.
	root string
	// Visitor Provider for generating the list of visitors.
	vp visitorProvider
}

// ParserOpt has often customized parser options. Use for example in NewParser.
type ParserOpt struct {
	// Vet turns on vetting mode, which catches a wider range of cross-cluster errors.
	Vet bool
	// Validate will raise validation errors if set.
	Validate bool
	// Bespin will enable the Bespin visitors.
	Bespin bool
}

// visitorProvider is an interface that abstracts out the source of visitors
// that are used to walk the AST.
type visitorProvider interface {
	visitors(apiInfo *meta.APIInfo,
		syncs []*v1alpha1.Sync,
		clusters []clusterregistry.Cluster,
		selectors []v1alpha1.ClusterSelector,
		opts ParserOpt) []ast.Visitor
}

// nomosVisitorProvider is the default visitor provider.  It handles
// plain vanilla nomos configs.
type nomosVisitorProvider struct{}

func (n nomosVisitorProvider) visitors(apiInfo *meta.APIInfo,
	syncs []*v1alpha1.Sync,
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector,
	opts ParserOpt) []ast.Visitor {

	specs := toInheritanceSpecs(syncs)
	visitors := []ast.Visitor{
		validation.NewInputValidator(syncs, specs, clusters, selectors, opts.Vet),
		transform.NewPathAnnotationVisitor(),
		validation.NewScope(apiInfo),
		transform.NewClusterSelectorVisitor(), // Filter out unneeded parts of the tree
		transform.NewAnnotationInlinerVisitor(),
		transform.NewInheritanceVisitor(specs),
	}
	if spec, found := specs[corev1.SchemeGroupVersion.WithKind("ResourceQuota").GroupKind()]; found && spec.Mode == v1alpha1.HierarchyModeHierarchicalQuota {
		visitors = append(visitors, transform.NewQuotaVisitor())
	}
	visitors = append(visitors, validation.NewNameValidator())

	return visitors
}

// bespinVisitorProvider is used when bespin is enabled to handle the bespin specific
// parts that won't pass the regular nomos checks.
type bespinVisitorProvider struct{}

func (b bespinVisitorProvider) visitors(apiInfo *meta.APIInfo,
	syncs []*v1alpha1.Sync,
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector,
	opts ParserOpt) []ast.Visitor {
	// TODO(b/119825336): Bespin and the InputValidator are having trouble playing
	// nicely. For now, just return the visitors that Bespin needs.
	return []ast.Visitor{
		transform.NewGCPHierarchyVisitor(),
		transform.NewGCPPolicyVisitor(),
		validation.NewScope(apiInfo),
		validation.NewNameValidator(),
	}
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
	var vp visitorProvider = nomosVisitorProvider{}
	if opts.Bespin {
		vp = bespinVisitorProvider{}
	}
	p := &Parser{
		opts:            opts,
		factory:         f,
		discoveryClient: dc,
		vp:              vp,
	}
	return p, nil
}

func toDirInfoMap(fileInfos []ast.FileObject) map[string][]ast.FileObject {
	result := make(map[string][]ast.FileObject)

	// If a directory has resources, its value in the map will be non-nil.
	for _, i := range fileInfos {
		d := filepath.Dir(i.Source)
		result[d] = append(result[d], i)
	}

	return result
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// * system/ (may be absent)
// * cluster/
// * clusterregistry/ (may be absent)
// * namespaces/ (recursively)
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

	// Special processing for <root>/system/*
	syncs := p.processSystemDir(filepath.Join(root, repo.SystemDir), fsCtx, apiInfo, &errorBuilder)
	if errorBuilder.HasErrors() {
		// Don't continue processing if any errors encountered processing system/
		return nil, errorBuilder.Build()
	}

	clusterDir := filepath.Join(root, repo.ClusterDir)
	clusterInfos := p.readResources(clusterDir, &errorBuilder)
	syntax.FlatDirectoryValidator.Validate(toSources(clusterInfos), &errorBuilder)
	validateObjects(clusterInfos, &errorBuilder)

	clusterregistryPath := filepath.Join(root, repo.ClusterRegistryDir)
	clusterregistryInfos := p.readResources(clusterregistryPath, &errorBuilder)
	syntax.FlatDirectoryValidator.Validate(toSources(clusterregistryInfos), &errorBuilder)
	validateObjects(clusterregistryInfos, &errorBuilder)

	nsDir := filepath.Join(root, repo.NamespacesDir)
	nsDirsOrdered := p.allDirs(nsDir, &errorBuilder)

	syntax.DirectoryNameValidator.Validate(nsDirsOrdered, &errorBuilder)
	semantic.DuplicateDirectoryValidator{Dirs: nsDirsOrdered}.Validate(&errorBuilder)

	nsInfos := p.readResources(nsDir, &errorBuilder)
	validateObjects(nsInfos, &errorBuilder)

	// TODO(filmil): dirInfos could just be map[string]ast.FileObject, it seems.  Let's wait
	// until the new repo format commit lands, and change it then.
	dirInfos := toDirInfoMap(nsInfos)
	policies, err := p.processDirs(apiInfo, dirInfos, clusterInfos, clusterregistryInfos, nsDirsOrdered, clusterDir, fsCtx, syncs)
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
		errorBuilder.Add(vet.MissingDirectoryError{})
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
			fileObject := ast.FileObject{Object: object, Source: source}
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
	clusterregistryObjects []ast.FileObject,
	nsDirsOrdered []string,
	clusterDir string,
	fsRoot *ast.Root,
	syncs []*v1alpha1.Sync) (*v1.AllPolicies, error) {

	errorBuilder := multierror.Builder{}
	namespaceDirs := make(map[string]bool)

	processClusterDir(clusterObjects, fsRoot)
	clusters, selectors := processClusterRegistryDir(clusterregistryObjects, &errorBuilder)
	cs, err := sel.NewClusterSelectors(clusters, selectors, os.Getenv("CLUSTER_NAME"))
	if err != nil {
		// TODO(b/120229144): To be factored into KNV Error.
		errorBuilder.Add(errors.Wrapf(err, "could not create cluster selectors"))
		return nil, errorBuilder.Build()
	}
	sel.SetClusterSelector(cs, fsRoot)

	treeGenerator := NewDirectoryTree()
	if len(nsDirsOrdered) > 0 {
		rootDir := nsDirsOrdered[0]
		infos := dirInfos[rootDir]
		processNamespacesDir(rootDir, infos, namespaceDirs, treeGenerator, true, &errorBuilder)
		if errorBuilder.HasErrors() {
			errorBuilder.Add(errors.Wrapf(err, "directory is invalid: %s", rootDir))
			return nil, errorBuilder.Build()
		}
		for _, d := range nsDirsOrdered[1:] {
			infos := dirInfos[d]
			processNamespacesDir(d, infos, namespaceDirs, treeGenerator, false, &errorBuilder)
			if errorBuilder.HasErrors() {
				errorBuilder.Add(errors.Wrapf(err, "directory is invalid: %s", d))
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

	visitors := p.vp.visitors(apiInfo, syncs, clusters, selectors, p.opts)

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
					if gk == rbacv1.SchemeGroupVersion.WithKind("RoleBinding").GroupKind() {
						effectiveMode = v1alpha1.HierarchyModeInherit
					} else if gk == corev1.SchemeGroupVersion.WithKind("ResourceQuota").GroupKind() {
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

func processClusterDir(
	objects []ast.FileObject,
	fsRoot *ast.Root) {
	for _, i := range objects {
		fsRoot.Cluster.Objects = append(fsRoot.Cluster.Objects, &ast.ClusterObject{FileObject: i})
	}
}

func processNamespacesDir(
	dir string,
	objects []ast.FileObject,
	namespaceDirs map[string]bool,
	treeGenerator *DirectoryTree,
	root bool, errorBuilder *multierror.Builder) {
	var treeNode *ast.TreeNode
	for _, object := range objects {
		switch object.Object.(type) {
		case *corev1.Namespace:
			namespaceDirs[dir] = true
			if root {
				treeNode = treeGenerator.SetRootDir(dir, ast.Namespace)
			} else {
				treeNode = treeGenerator.AddDir(dir, ast.Namespace)
			}
			processNamespaceDir(objects, treeNode, errorBuilder)
			return
		}
	}
	// No namespace resource was found.
	if root {
		treeNode = treeGenerator.SetRootDir(dir, ast.AbstractNamespace)
	} else {
		treeNode = treeGenerator.AddDir(dir, ast.AbstractNamespace)
	}

	for _, i := range objects {
		switch o := i.Object.(type) {
		case *v1alpha1.NamespaceSelector:
			treeNode.Selectors[o.Name] = o
		default:
			treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.FileObject{Object: o, Source: i.Source}})
		}
	}
}

func processNamespaceDir(objects []ast.FileObject, treeNode *ast.TreeNode, errorBuilder *multierror.Builder) {
	syntax.NamespacesKindValidator.Validate(objects, errorBuilder)

	for _, object := range objects {
		gvk := object.GroupVersionKind()
		if gvk == corev1.SchemeGroupVersion.WithKind("Namespace") {
			// TODO: Move this out.
			metaObj := object.Object.(metav1.Object)
			treeNode.Labels = metaObj.GetLabels()
			treeNode.Annotations = metaObj.GetAnnotations()
			continue
		}

		treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.FileObject{Object: object.Object, Source: object.Source}})
	}

}

// processSystemDir processes resources in system dir including:
// - Nomos Repo
// - Reserved Namespaces
// - Syncs
func (p *Parser) processSystemDir(systemDir string, fsRoot *ast.Root,
	apiInfo *meta.APIInfo, errorBuilder *multierror.Builder) []*v1alpha1.Sync {
	// Ignore individual file read errors for now and continue processing parsed files.
	objects := p.readRequiredResources(systemDir, errorBuilder)
	validateObjects(objects, errorBuilder)

	syntax.FlatDirectoryValidator.Validate(toSources(objects), errorBuilder)
	syntax.RepoVersionValidator.Validate(objects, errorBuilder)
	syntax.SystemKindValidator.Validate(objects, errorBuilder)
	semantic.RepoCountValidator{Objects: objects}.Validate(errorBuilder)
	semantic.ConfigMapCountValidator{Objects: objects}.Validate(errorBuilder)

	syncMap := make(map[string][]*v1alpha1.Sync)
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.Repo:
			fsRoot.Repo = o

		case *corev1.ConfigMap:
			fsRoot.ReservedNamespaces = &ast.ReservedNamespaces{ConfigMap: *o}

		case *v1alpha1.Sync:
			syncMap[object.Source] = append(syncMap[object.Source], o)
		}
	}

	for source, syncs := range syncMap {
		for _, sync := range syncs {
			for _, group := range sync.Spec.Groups {
				for k := range group.Kinds {
					// TODO(willbeason) Ensure globally that Kinds with the same Group are not given multiple versions.
					validateSyncKind(sync.Name, &group.Kinds[k], group.Group, fsRoot.Repo.Spec.ExperimentalInheritance, source, apiInfo, errorBuilder)
				}
			}
		}
	}

	var allSyncs []*v1alpha1.Sync
	for _, syncs := range syncMap {
		allSyncs = append(allSyncs, syncs...)
	}
	if p.opts.Bespin {
		allSyncs = append(allSyncs, bespinv1.Syncs...)
	}
	return allSyncs
}

// validateSyncKind validates a SyncKind.
// - name is the Sync name
// - kind is the SyncKind to be validated
// - group is the group of the enclosing SyncGroup
// - inheritance is the value of the experimentalInheritance flag in Repo.
// - source is the source file of the Sync, used for error messages.
func validateSyncKind(name string, kind *v1alpha1.SyncKind, group string, inheritance bool,
	source string, apiInfo *meta.APIInfo, errorBuilder *multierror.Builder) {
	if len(kind.Versions) > 1 {
		errorBuilder.Add(vet.MultipleVersionForSameSyncedTypeError{Source: source, Kind: *kind})
	}
	if kind.Kind == string(ast.Namespace) && group == "" {
		// Require that Group is the empty string since that is the core Namespace object
		// A different group could validly define a Kind called "Namespace"
		errorBuilder.Add(vet.IllegalNamespaceSyncDeclarationError{Source: source})
	}
	syncGVK := schema.GroupVersionKind{
		Group:   group,
		Version: kind.Versions[0].Version,
		Kind:    kind.Kind,
	}
	if unsupportedSyncResources[syncGVK] || syncGVK.Group == policyhierarchy.GroupName {
		errorBuilder.Add(vet.UnsupportedResourceInSyncError{
			SyncPath:     source,
			ResourceType: syncGVK,
		})
	} else if !apiInfo.Exists(syncGVK) {
		errorBuilder.Add(vet.UnknownResourceInSyncError{
			SyncPath:     source,
			ResourceType: syncGVK,
		})
	}
	if inheritance {
		if kind.Kind == "ResourceQuota" && group == "" {
			checkModeAllowed([]v1alpha1.HierarchyModeType{v1alpha1.HierarchyModeDefault, v1alpha1.HierarchyModeHierarchicalQuota, v1alpha1.HierarchyModeInherit, v1alpha1.HierarchyModeNone}, kind.HierarchyMode, name, errorBuilder)
		} else {
			checkModeAllowed([]v1alpha1.HierarchyModeType{v1alpha1.HierarchyModeDefault, v1alpha1.HierarchyModeInherit, v1alpha1.HierarchyModeNone}, kind.HierarchyMode, name, errorBuilder)
		}
	} else {
		checkModeAllowed([]v1alpha1.HierarchyModeType{v1alpha1.HierarchyModeDefault}, kind.HierarchyMode, name, errorBuilder)
	}
}

func checkModeAllowed(allowed []v1alpha1.HierarchyModeType, actual v1alpha1.HierarchyModeType, name string, eb *multierror.Builder) {
	if !containsMode(allowed, actual) {
		eb.Add(vet.IllegalHierarchyModeError{Name: name, Mode: actual, Allowed: allowed})
	}
}

func containsMode(haystack []v1alpha1.HierarchyModeType, needle v1alpha1.HierarchyModeType) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// processClusterRegistryDir looks at all files in <root>/clusterregistry and
// extracts Cluster and ClusterSelector objects out. dirname is the directory
// name relative to the root directory of the repository, and infos is the set
// of resource data that were read from the directory.
func processClusterRegistryDir(objects []ast.FileObject, errorBuilder *multierror.Builder) ([]clusterregistry.Cluster, []v1alpha1.ClusterSelector) {
	syntax.ClusterregistryKindValidator.Validate(objects, errorBuilder)

	var crc []clusterregistry.Cluster
	var css []v1alpha1.ClusterSelector
	for _, object := range objects {
		switch o := object.Object.(type) {
		case *v1alpha1.ClusterSelector:
			css = append(css, *o)
		case *clusterregistry.Cluster:
			crc = append(crc, *o)
		}
	}

	return crc, css
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

func validateObjects(objects []ast.FileObject, errorBuilder *multierror.Builder) {
	syntax.AnnotationValidator.Validate(objects, errorBuilder)
	syntax.LabelValidator.Validate(objects, errorBuilder)
	syntax.MetadataNamespaceValidator.Validate(objects, errorBuilder)
	syntax.MetadataNameValidator.Validate(objects, errorBuilder)
	syntax.SystemOnlyResourceValidator.Validate(objects, errorBuilder)

	semantic.ConflictingResourceQuotaValidator{Objects: objects}.Validate(errorBuilder)
	semantic.DuplicateNamespaceValidator{Objects: objects}.Validate(errorBuilder)
	semantic.DuplicateNameValidator{Objects: objects}.Validate(errorBuilder)
}

func toSources(infos []ast.FileObject) []string {
	result := make([]string, len(infos))
	for i, info := range infos {
		result[i] = info.Source
	}
	return result
}
