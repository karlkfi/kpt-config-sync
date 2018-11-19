// Reviewed by sunilarora
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
	"path"
	"path/filepath"
	"sort"
	"strings"

	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/backend"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/policyimporter/meta"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/google/nomos/pkg/util/namespaceutil"
	policynodevalidator "github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions/resource"
	kubevalidation "k8s.io/kubernetes/pkg/kubectl/validation"
)

func init() {
	// Add Nomos and Bespin types to the Scheme used by util.AsDefaultVersionedOrOriginal for
	// converting Unstructured to specific types.
	runtime.Must(v1.AddToScheme(legacyscheme.Scheme))
	runtime.Must(v1alpha1.AddToScheme(legacyscheme.Scheme))
	runtime.Must(bespinv1.AddToScheme(legacyscheme.Scheme))
	runtime.Must(clusterregistry.AddToScheme(legacyscheme.Scheme))
}

// Parser reads files on disk and builds Nomos CRDs.
type Parser struct {
	opts            ParserOpt
	factory         cmdutil.Factory
	discoveryClient discovery.ServerResourcesInterface
	// OS-specific path to root.
	root string
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

// NewParser creates a new Parser.
// clientConfig can be used to configure api server client. It should be set to nil when running in cluster.
// resources is the list returned by the DisoveryClient ServerResources call which represents resources
// 		that are returned by the API server during discovery.
// opts turns on options for the parser.
// TODO(118887045): Don't pass in a discoveryClient. Just use the one in from RestClientGetter.ToDiscoveryClient().
func NewParser(clientGetter genericclioptions.RESTClientGetter, discoveryClient discovery.ServerResourcesInterface, opts ParserOpt) (*Parser, error) {
	return NewParserWithFactory(cmdutil.NewFactory(clientGetter), discoveryClient, opts)
}

// NewParserWithFactory creates a new Parser using the specified factory.
// NewParser is the more common constructor, but this is useful for testing.
func NewParserWithFactory(f cmdutil.Factory, dc discovery.ServerResourcesInterface, opts ParserOpt) (*Parser, error) {
	p := &Parser{
		opts:            opts,
		factory:         f,
		discoveryClient: dc,
	}
	return p, nil
}

func toDirInfoMap(fileInfos []*resource.Info) map[string][]*resource.Info {
	result := make(map[string][]*resource.Info)

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
	clusterInfos := p.readResources(clusterDir, false, &errorBuilder)
	p.validateDuplicateNames(toDirInfoMap(clusterInfos), &errorBuilder)

	clusterregistryPath := filepath.Join(root, repo.ClusterRegistryDir)
	clusterregistryInfos := p.readResources(clusterregistryPath, false, &errorBuilder)
	p.validateDuplicateNames(toDirInfoMap(clusterregistryInfos), &errorBuilder)

	nsDir := filepath.Join(root, repo.NamespacesDir)
	nsDirsOrdered := allDirs(nsDir, &errorBuilder)

	p.validateDirNames(root, nsDirsOrdered, &errorBuilder)

	fileInfos := p.readResources(nsDir, true, &errorBuilder)

	// TODO(filmil): dirInfos could just be map[string]runtime.Object, it seems.  Let's wait
	// until the new repo format commit lands, and change it then.
	dirInfos := toDirInfoMap(fileInfos)
	p.validateDuplicateNames(dirInfos, &errorBuilder)

	policies, err := p.processDirs(apiInfo, dirInfos, clusterInfos, clusterregistryInfos, nsDirsOrdered,
		clusterDir, fsCtx, syncs)
	if err != nil {
		errorBuilder.Add(err)
		return nil, errorBuilder.Build()
	}
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
func (p *Parser) readRequiredResources(dir string, allowSubdirectories bool, errorBuilder *multierror.Builder) []*resource.Info {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		errorBuilder.Add(validation.MissingDirectoryError{})
		return nil
	}
	return p.readResources(dir, allowSubdirectories, errorBuilder)
}

// readResources walks dir recursively, looking for resources, and builds FileInfos from them.
func (p *Parser) readResources(dir string, allowSubdirectories bool, errorBuilder *multierror.Builder) []*resource.Info {
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

	var fileInfos []*resource.Info
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
		fileInfos, err = result.Infos()
		if err != nil {
			errorBuilder.Add(err)
		}

		if !allowSubdirectories {
			// If subdirectories are not allowed, return an error for each invalid subdirectory.
			for _, info := range fileInfos {
				if subDir := path.Dir(info.Source); subDir != dir {
					errorBuilder.Add(validation.IllegalSubdirectoryError{Dir: dir, SubDir: subDir})
				}
			}
		}
	}
	return fileInfos
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
	dirInfos map[string][]*resource.Info,
	clusterInfos []*resource.Info,
	clusterregistryInfos []*resource.Info,
	nsDirsOrdered []string,
	clusterDir string,
	fsCtx *ast.Root,
	syncs []*v1alpha1.Sync) (*v1.AllPolicies, error) {

	errorBuilder := multierror.Builder{}
	namespaceDirs := make(map[string]bool)

	if err := p.processClusterDir(clusterDir, clusterInfos, fsCtx); err != nil {
		errorBuilder.Add(errors.Wrapf(err, "cluster directory is invalid: %s", clusterDir))
		return nil, errorBuilder.Build()
	}
	clusters, selectors := p.processClusterRegistryDir(repo.ClusterRegistryDir, clusterregistryInfos, &errorBuilder)
	cs, err := sel.NewClusterSelectors(clusters, selectors, os.Getenv("CLUSTER_NAME"))
	if err != nil {
		errorBuilder.Add(errors.Wrapf(err, "could not create cluster selectors"))
		return nil, errorBuilder.Build()
	}
	sel.SetClusterSelector(cs, fsCtx)

	treeGenerator := NewDirectoryTree()
	if len(nsDirsOrdered) > 0 {
		rootDir := nsDirsOrdered[0]
		infos := dirInfos[rootDir]
		if err = p.processNamespacesDir(rootDir, infos, namespaceDirs, treeGenerator, true); err != nil {
			errorBuilder.Add(errors.Wrapf(err, "directory is invalid: %s", rootDir))
			return nil, errorBuilder.Build()
		}
		for _, d := range nsDirsOrdered[1:] {
			infos := dirInfos[d]
			if err = p.processNamespacesDir(d, infos, namespaceDirs, treeGenerator, false); err != nil {
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
	fsCtx.Tree = tree

	visitors, err := buildVisitors(apiInfo, toAllowedGVKs(syncs), clusters, selectors, p.opts)
	if err != nil {
		errorBuilder.Add(err)
		return nil, errorBuilder.Build()
	}

	for _, visitor := range visitors {
		fsCtx = fsCtx.Accept(visitor).(*ast.Root)
		if err := visitor.Error(); err != nil {
			errorBuilder.Add(err)
			return nil, errorBuilder.Build()
		}
	}

	outputVisitor := backend.NewOutputVisitor(syncs)
	fsCtx.Accept(outputVisitor)
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

func buildVisitors(apiInfo *meta.APIInfo,
	allowedGVKs map[schema.GroupVersionKind]bool,
	clusters []clusterregistry.Cluster,
	selectors []v1alpha1.ClusterSelector,
	opts ParserOpt) ([]ast.CheckingVisitor, error) {

	var visitors []ast.CheckingVisitor
	// TODO(b/119825336): This visitor shouldn't come before the validator,
	// but the validator isn't allowing this to go through.
	if opts.Bespin {
		visitors = append(visitors, transform.NewGCPHierarchyVisitor(), transform.NewGCPPolicyVisitor())
	}

	visitors = append(visitors,
		validation.NewInputValidator(allowedGVKs, clusters, selectors, opts.Vet),
		transform.NewPathAnnotationVisitor(),
		validation.NewScope(apiInfo),
		transform.NewClusterSelectorVisitor(), // Filter out unneeded parts of the tree
		transform.NewAnnotationInlinerVisitor(),
		transform.NewInheritanceVisitor(
			[]transform.InheritanceSpec{
				{
					GroupVersionKind: rbacv1.SchemeGroupVersion.WithKind("RoleBinding"),
				},
			},
		),
		transform.NewQuotaVisitor(),
		validation.NewNameValidator(),
	)

	return visitors, nil
}

func toAllowedGVKs(syncs []*v1alpha1.Sync) map[schema.GroupVersionKind]bool {
	allowedGVKs := make(map[schema.GroupVersionKind]bool)
	for _, sync := range syncs {
		for _, sg := range sync.Spec.Groups {
			for _, k := range sg.Kinds {
				for _, v := range k.Versions {
					gvk := schema.GroupVersionKind{Group: sg.Group, Kind: k.Kind, Version: v.Version}
					allowedGVKs[gvk] = true
				}
			}
		}
	}
	return allowedGVKs
}

func (p *Parser) processClusterDir(
	dir string,
	infos []*resource.Info,
	fsCtx *ast.Root) error {
	for _, i := range infos {
		o := cmdutil.AsDefaultVersionedOrOriginal(i.Object, i.Mapping)
		fsCtx.Cluster.Objects = append(fsCtx.Cluster.Objects, &ast.ClusterObject{FileObject: ast.FileObject{Object: o, Source: p.relativePath(i.Source)}})
	}

	return nil
}

func (p *Parser) processNamespacesDir(
	dir string,
	infos []*resource.Info,
	namespaceDirs map[string]bool,
	treeGenerator *DirectoryTree,
	root bool) error {
	var treeNode *ast.TreeNode
	for _, i := range infos {
		o := cmdutil.AsDefaultVersionedOrOriginal(i.Object, i.Mapping)

		switch o.(type) {
		case *corev1.Namespace:
			namespaceDirs[dir] = true
			if root {
				treeNode = treeGenerator.SetRootDir(dir, ast.Namespace)
			} else {
				treeNode = treeGenerator.AddDir(dir, ast.Namespace)
			}
			return p.processNamespaceDir(dir, infos, treeNode)
		}
	}
	// No namespace resource was found.
	if root {
		treeNode = treeGenerator.SetRootDir(dir, ast.AbstractNamespace)
	} else {
		treeNode = treeGenerator.AddDir(dir, ast.AbstractNamespace)
	}

	for _, i := range infos {
		obj := cmdutil.AsDefaultVersionedOrOriginal(i.Object, i.Mapping)
		switch o := obj.(type) {
		case *v1alpha1.NamespaceSelector:
			treeNode.Selectors[o.Name] = o
		default:
			treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.FileObject{Object: o, Source: p.relativePath(i.Source)}})
		}
	}
	return nil
}

func (p *Parser) processNamespaceDir(dir string, infos []*resource.Info, treeNode *ast.TreeNode) error {
	v := newValidator()

	for _, i := range infos {
		o := cmdutil.AsDefaultVersionedOrOriginal(i.Object, i.Mapping)
		gvk := o.GetObjectKind().GroupVersionKind()
		if gvk == corev1.SchemeGroupVersion.WithKind("Namespace") {
			// TODO: Move this out.
			metaObj := o.(metav1.Object)
			treeNode.Labels = metaObj.GetLabels()
			treeNode.Annotations = metaObj.GetAnnotations()
			v.MarkSeen(gvk)
			continue
		}

		if o.GetObjectKind().GroupVersionKind() == v1alpha1.SchemeGroupVersion.WithKind("NamespaceSelector") {
			v.ObjectDisallowedInContext(i, o.GetObjectKind().GroupVersionKind())
		}
		if v.err != nil {
			return v.err
		}

		treeNode.Objects = append(treeNode.Objects, &ast.NamespaceObject{FileObject: ast.FileObject{Object: o, Source: p.relativePath(i.Source)}})
	}

	v.HaveSeen(schema.GroupVersionKind{Version: "v1", Kind: "Namespace"})
	if v.err != nil {
		return v.err
	}

	return nil
}

// processSystemDir processes resources in system dir including:
// - Nomos Repo
// - Reserved Namespaces
// - Syncs
func (p *Parser) processSystemDir(systemDir string, fsCtx *ast.Root,
	apiInfo *meta.APIInfo, errorBuilder *multierror.Builder) []*v1alpha1.Sync {
	// Ignore individual file read errors for now and continue processing parsed files.
	fileInfos := p.readRequiredResources(systemDir, false, errorBuilder)
	p.validateDuplicateNames(toDirInfoMap(fileInfos), errorBuilder)

	var syncs []*v1alpha1.Sync
	repos := make(map[*v1alpha1.Repo]string)         // holds all Repo definitions
	configMaps := make(map[*corev1.ConfigMap]string) // holds all ConfigMap definitions
	for _, info := range fileInfos {
		obj := cmdutil.AsDefaultVersionedOrOriginal(info.Object, info.Mapping)

		switch o := obj.(type) {
		case *v1alpha1.Repo:
			repos[o] = info.Source
			if version := o.Spec.Version; version != "0.1.0" {
				errorBuilder.Add(validation.UnsupportedRepoSpecVersion{Source: info.Source, Name: o.Name, Version: version})
			}
			fsCtx.Repo = o

		case *corev1.ConfigMap:
			configMaps[o] = info.Source
			fsCtx.ReservedNamespaces = &ast.ReservedNamespaces{ConfigMap: *o}

		case *v1alpha1.Sync:
			sync := o

			// We only support one version per synced type
			for _, group := range sync.Spec.Groups {
				for _, kind := range group.Kinds {
					if len(kind.Versions) > 1 {
						errorBuilder.Add(validation.MultipleVersionForSameSyncedTypeError{Source: info.Source, Group: group, Kind: kind})
					}
					if kind.Kind == string(ast.Namespace) && group.Group == "" {
						// Require that Group is the empty string since that is the core Namespace object
						// A different group could validly define a Kind called "Namespace"
						errorBuilder.Add(validation.IllegalNamespaceSyncDeclarationError{Source: info.Source})
					}
					syncGVK := schema.GroupVersionKind{
						Group:   group.Group,
						Version: kind.Versions[0].Version,
						Kind:    kind.Kind,
					}
					if !apiInfo.Exists(syncGVK) {
						errorBuilder.Add(validation.UnknownResourceInSyncError{
							SyncPath:     p.relativePath(info.Source),
							ResourceType: syncGVK,
						})
					}
				}
			}

			syncs = append(syncs, sync)
		default:
			errorBuilder.Add(validation.IllegalSystemObjectDefinitionInSystemError{Source: info.Source, GroupVersionKind: o.GetObjectKind().GroupVersionKind()})
		}
	}

	if len(repos) == 0 {
		errorBuilder.Add(validation.MissingRepoError{})
	} else if len(repos) >= 2 {
		errorBuilder.Add(validation.MultipleRepoDefinitionsError{Repos: repos})
	}

	if len(configMaps) >= 2 {
		errorBuilder.Add(validation.MultipleConfigMapsError{ConfigMaps: configMaps})
	}

	return syncs
}

// processClusterRegistryDir looks at all files in <root>/clusterregistry and
// extracts Cluster and ClusterSelector objects out. dirname is the directory
// name relative to the root directory of the repository, and infos is the set
// of resource data that were read from the directory.
func (p *Parser) processClusterRegistryDir(dirname string, infos []*resource.Info, errorBuilder *multierror.Builder) ([]clusterregistry.Cluster, []v1alpha1.ClusterSelector) {
	v := newValidator()
	var crc []clusterregistry.Cluster
	var css []v1alpha1.ClusterSelector
	for _, i := range infos {
		obj := cmdutil.AsDefaultVersionedOrOriginal(i.Object, i.Mapping)
		switch o := obj.(type) {
		case *v1alpha1.ClusterSelector:
			css = append(css, *o)
		case *clusterregistry.Cluster:
			crc = append(crc, *o)
		default:
			// No other objects are allowed in the clusterregistry directory.
			v.ObjectDisallowedInContext(i, o.GetObjectKind().GroupVersionKind())
		}
	}

	if v.err != nil {
		errorBuilder.Add(errors.Wrapf(v.err, "clusterregistry directory is invalid: %s", dirname))
	}
	return crc, css
}

// allDirs returns absolute paths of all directories in root, in lexicographic (depth-first) order.
func allDirs(root string, errorBuilder *multierror.Builder) []string {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		errorBuilder.Add(err)
		return nil
	}
	return paths
}

// validateDirNames validates that:
// 1. Directory name is not reserved by the system.
// 2. Directory name is a valid namespace name:
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
func (p *Parser) validateDirNames(root string, dirs []string, errorBuilder *multierror.Builder) {
	dirNames := make(map[string][]string)
	for _, d := range dirs {
		n := filepath.Base(d)
		if namespaceutil.IsInvalid(n) {
			errorBuilder.Add(validation.InvalidDirectoryNameError{Dir: d})
		}
		if namespaceutil.IsReserved(n) {
			errorBuilder.Add(validation.ReservedDirectoryNameError{Dir: d})
		}
		relpath, _ := filepath.Rel(root, d)
		if names, ok := dirNames[n]; ok {
			dirNames[n] = append(names, relpath)
		} else {
			dirNames[n] = []string{relpath}
		}
	}

	for _, duplicates := range dirNames {
		if len(duplicates) > 1 {
			errorBuilder.Add(validation.DuplicateDirectoryNameError{Duplicates: duplicates})
		}
	}
}

func (p *Parser) validateDuplicateNames(dirInfos map[string][]*resource.Info, errorBuilder *multierror.Builder) {
	seenObjectNames := make(map[schema.GroupVersionKind]map[string][]*resource.Info)
	seenNamespaceDirs := make(map[string][]*resource.Info)
	seenResourceQuotas := make(map[string][]*resource.Info)
	seenDirs := make(map[string]map[string]struct{})

	for _, infos := range dirInfos {
		for _, info := range infos {
			source := p.relativePath(info.Source)
			dir := path.Dir(source)

			if info.Namespace != "" {
				errorBuilder.Add(validation.IllegalMetadataNamespaceDeclarationError{Root: p.root, Info: info})
			}

			gvk := info.Mapping.GroupVersionKind
			if info.Name == "" {
				errorBuilder.Add(validation.MissingObjectNameError{Source: source, Info: info})
				continue
			}

			if _, found := seenDirs[path.Base(dir)]; !found {
				seenDirs[path.Base(dir)] = map[string]struct{}{dir: {}}
			} else {
				seenDirs[path.Base(dir)][dir] = struct{}{}
			}

			switch gvk {
			case corev1.SchemeGroupVersion.WithKind("Namespace"):
				if dir == repo.NamespacesDir {
					errorBuilder.Add(validation.IllegalTopLevelNamespaceError{Source: source, Info: info})
					continue
				} else if path.Base(dir) != info.Name {
					errorBuilder.Add(validation.InvalidNamespaceNameError{Source: source, Expected: path.Base(dir), Actual: info.Name})
				}
				seenNamespaceDirs[dir] = append(seenNamespaceDirs[dir], info)
			case corev1.SchemeGroupVersion.WithKind("ResourceQuota"):
				seenResourceQuotas[dir] = append(seenResourceQuotas[dir], info)
			default:
				if _, found := seenObjectNames[gvk]; !found {
					seenObjectNames[gvk] = make(map[string][]*resource.Info)
				}
				seenObjectNames[gvk][info.Name] = append(seenObjectNames[gvk][info.Name], info)
			}
		}
	}

	// Check for namespace object collisions
	for _, namespaces := range seenNamespaceDirs {
		if len(namespaces) > 1 {
			errorBuilder.Add(validation.MultipleNamespacesError{Root: p.root, Duplicates: namespaces})
		}
	}

	// Check for ResourceQuota collisions
	for dir, quotas := range seenResourceQuotas {
		if len(quotas) > 1 {
			errorBuilder.Add(validation.ConflictingResourceQuotaError{Path: dir, Duplicates: quotas})
		}
	}

	// Check for directory name collisions
	for _, paths := range seenDirs {
		if len(paths) > 1 {
			duplicates := []string{}
			for aPath := range paths {
				duplicates = append(duplicates, aPath)
			}
			errorBuilder.Add(validation.DuplicateDirectoryNameError{Duplicates: duplicates})
		}
	}

	// Check for object name collisions
	for _, objectsByNames := range seenObjectNames {
		// All objects have the same kind
		for name, objects := range objectsByNames {
			// All objects have the same name and kind
			sort.Slice(objects, func(i, j int) bool {
				// Sort by source file
				return path.Dir(objects[i].Source) < path.Dir(objects[j].Source)
			})

			for i := 0; i < len(objects); {
				dir := p.relativePath(path.Dir(objects[i].Source))
				duplicates := []*resource.Info{objects[i]}

				for j := i + 1; j < len(objects); j++ {
					if strings.HasPrefix(p.relativePath(objects[j].Source), dir) {
						// Pick up duplicates in the same directory and child directories.
						duplicates = append(duplicates, objects[j])
					} else {
						// Since objects are sorted by paths, this guarantees that objects within a directory
						// will be contiguous. We can exit at the first non-matching source path.
						break
					}
				}

				if len(duplicates) > 1 {
					errorBuilder.Add(validation.ObjectNameCollisionError{Name: name, RootPath: p.root, Duplicates: duplicates})
				}

				// Recall that len(duplicates) is always at least 1.
				// There's no need to have multiple errors when more than two objects collide.
				i += len(duplicates)
			}
		}
	}
}
