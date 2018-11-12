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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/blang/semver"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/backend"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	sel "github.com/google/nomos/pkg/policyimporter/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/multierror"
	"github.com/google/nomos/pkg/util/namespaceutil"
	policynodevalidator "github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
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

	// Special processing for <root>/system/*
	var syncs []*v1alpha1.Sync
	systemDir := filepath.Join(root, repo.SystemDir)
	if files, err := ioutil.ReadDir(systemDir); err == nil {
		if len(files) > 0 {
			// The system directory must at least have one file, defining the Repo object.
			if syncs, err = p.processSystemDir(root, fsCtx); err != nil {
				return nil, err
			}
		} else {
			return nil, validation.EmptySystemDirectoryError{}
		}
	} else if os.IsNotExist(err) {
		return nil, validation.MissingSystemDirectoryError{}
	} else {
		return nil, errors.Wrapf(err, "while checking existence of system directory")
	}

	clusterDir := filepath.Join(root, repo.ClusterDir)
	clusterInfos, err := p.readResources(clusterDir, true)
	if err != nil {
		return nil, err
	}

	var clusterregistryInfos []*resource.Info
	clusterregistryPath := filepath.Join(root, repo.ClusterRegistryDir)
	if _, err = os.Stat(clusterregistryPath); err == nil {
		clusterregistryInfos, err = p.readResources(clusterregistryPath, false)
		if err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		// It's OK not to define the clusterregistry directory, you just won't get
		// the PCA features.
		return nil, errors.Wrapf(err, "while checking existence of clusterregistry directory")
	}

	nsDir := filepath.Join(root, repo.NamespacesDir)
	nsDirsOrdered, err := allDirs(nsDir)
	if err != nil {
		return nil, err
	}
	if err = validateDirNames(nsDirsOrdered); err != nil {
		return nil, err
	}
	fileInfos, err := p.readResources(nsDir, true)
	if err != nil {
		return nil, err
	}

	// TODO(filmil): dirInfos could just be map[string]runtime.Object, it seems.  Let's wait
	// until the new repo format commit lands, and change it then.
	dirInfos := make(map[string][]*resource.Info)
	// Value of all dirs is initially set to nil.
	for _, d := range nsDirsOrdered {
		dirInfos[d] = nil
	}
	// If a directory has resources, its value in the map
	// will be non-nil.
	for _, i := range fileInfos {
		d := filepath.Dir(i.Source)
		dirInfos[d] = append(dirInfos[d], i)
	}

	if err := validateDuplicateNames(dirInfos); err != nil {
		return nil, err
	}

	resources, discoveryErr := p.discoveryClient.ServerResources()
	if discoveryErr != nil {
		return nil, errors.Wrap(discoveryErr, "failed to get server resources")
	}

	return p.processDirs(resources, dirInfos, clusterInfos, clusterregistryInfos, nsDirsOrdered,
		clusterDir, fsCtx, syncs)
}

func (p *Parser) relativePath(source string) string {
	r, err := filepath.Rel(p.root, source)
	if err != nil {
		panic(errors.Wrap(err, "programmer error"))
	}
	return r
}

// readResources walks dir recursively, looking for resources, and builds FileInfos from them.
func (p *Parser) readResources(dir string, recursive bool) ([]*resource.Info, error) {
	// If there aren't any resources, skip builder, because builder treats that as an error.
	var fileInfos []*resource.Info
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fileInfos, nil
	}
	visitors, err := resource.ExpandPathsToFileVisitors(
		nil, dir, true, resource.FileExtensions, kubevalidation.NullSchema{})
	if err != nil {
		return nil, err
	}
	if len(visitors) > 0 {
		s, err := p.factory.Validator(p.opts.Validate)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get schema")
		}
		result := p.factory.NewBuilder().
			Unstructured().
			Schema(s).
			ContinueOnError().
			FilenameParam(false, &resource.FilenameOptions{Recursive: recursive, Filenames: []string{dir}}).
			Do()
		fileInfos, err = result.Infos()

		if err != nil {
			errs := err.(utilerrors.Aggregate)
			return nil, errors.Wrapf(multierror.From(errs.Errors()),
				"failed to read resources from directory:\n%s", dir)
		}
	}
	return fileInfos, nil
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
func (p *Parser) processDirs(resources []*metav1.APIResourceList,
	dirInfos map[string][]*resource.Info,
	clusterInfos []*resource.Info,
	clusterregistryInfos []*resource.Info,
	nsDirsOrdered []string,
	clusterDir string,
	fsCtx *ast.Root,
	syncs []*v1alpha1.Sync) (*v1.AllPolicies, error) {
	namespaceDirs := make(map[string]bool)

	if err := p.processClusterDir(clusterDir, clusterInfos, fsCtx); err != nil {
		return nil, errors.Wrapf(err, "cluster directory is invalid: %s", clusterDir)
	}
	clusters, selectors, err := p.processClusterRegistryDir(repo.ClusterRegistryDir, clusterregistryInfos)
	if err != nil {
		return nil, errors.Wrapf(err, "clusterregistry directory is invalid: %s", clusterDir)
	}
	cs, err := sel.NewClusterSelectors(clusters, selectors, os.Getenv("CLUSTER_NAME"))
	if err != nil {
		return nil, errors.Wrapf(err, "could not create cluster selectors")
	}
	sel.SetClusterSelector(cs, fsCtx)

	treeGenerator := NewDirectoryTree()
	if len(nsDirsOrdered) > 0 {
		rootDir := nsDirsOrdered[0]
		infos := dirInfos[rootDir]
		if err = p.processNamespacesDir(rootDir, infos, namespaceDirs, treeGenerator, true); err != nil {
			return nil, errors.Wrapf(err, "directory is invalid: %s", rootDir)
		}
		for _, d := range nsDirsOrdered[1:] {
			infos := dirInfos[d]
			if err = p.processNamespacesDir(d, infos, namespaceDirs, treeGenerator, false); err != nil {
				return nil, errors.Wrapf(err, "directory is invalid: %s", d)
			}
		}
	}

	tree, err := treeGenerator.Build()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to treeify policy nodes")
	}
	fsCtx.Tree = tree

	scopeValidator, err := validation.NewScope(resources)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create scope validator")
	}

	visitors := []ast.CheckingVisitor{
		validation.NewInputValidator(toAllowedGVKs(syncs), clusters, selectors, p.opts.Vet),
		transform.NewPathAnnotationVisitor(),
		scopeValidator,
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
	}

	for _, visitor := range visitors {
		fsCtx = fsCtx.Accept(visitor).(*ast.Root)
		if err := visitor.Error(); err != nil {
			return nil, err
		}
	}

	outputVisitor := backend.NewOutputVisitor(syncs)
	fsCtx.Accept(outputVisitor)
	policies := outputVisitor.AllPolicies()

	if err := clusterpolicy.Validate(policies.ClusterPolicy); err != nil {
		return nil, err
	}
	v := policynodevalidator.FromMap(policies.PolicyNodes)
	if err := v.Validate(); err != nil {
		return nil, err
	}

	return policies, nil
}

func toAllowedGVKs(syncs []*v1alpha1.Sync) map[schema.GroupVersionKind]struct{} {
	allowedGVKs := make(map[schema.GroupVersionKind]struct{})
	for _, sync := range syncs {
		for _, sg := range sync.Spec.Groups {
			for _, k := range sg.Kinds {
				for _, v := range k.Versions {
					gvk := schema.GroupVersionKind{Group: sg.Group, Kind: k.Kind, Version: v.Version}
					allowedGVKs[gvk] = struct{}{}
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
			treeNode.Data = treeNode.Data.Add(validation.MetadataNameKey, i.Name)
			treeNode.Data = treeNode.Data.Add(validation.NamespaceSourceKey, p.relativePath(i.Source))
			v.HaveNotSeen(gvk).MarkSeen(gvk)
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
func (p *Parser) processSystemDir(root string, fsCtx *ast.Root) ([]*v1alpha1.Sync, error) {
	validator, err := p.factory.Validator(p.opts.Validate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get schema")
	}
	result := p.factory.NewBuilder().
		Unstructured().
		Schema(validator).
		ContinueOnError().
		FilenameParam(false, &resource.FilenameOptions{Recursive: false, Filenames: []string{filepath.Join(root, repo.SystemDir)}}).
		Do()
	fileInfos, err := result.Infos()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read resources from system dir")
	}
	v := newValidator()
	var syncs []*v1alpha1.Sync
	for _, i := range fileInfos {
		obj := cmdutil.AsDefaultVersionedOrOriginal(i.Object, i.Mapping)

		// Types in scope then alphabetical order.
		switch o := obj.(type) {
		case *v1alpha1.Repo:
			if _, err := semver.Parse(o.Spec.Version); err != nil {
				return nil, errors.Wrapf(err, "invalid semantic version %s. "+
					"Repo.Spec.Version must follow semantic versioning rules at http://semver.org",
					o.Spec.Version)
			}
			fsCtx.Repo = o

		case *corev1.ConfigMap:
			if o.Name != v1alpha1.ReservedNamespacesConfigMapName {
				return nil, errors.Errorf("invalid configmap in system dir %#v", o)
			}
			fsCtx.ReservedNamespaces = &ast.ReservedNamespaces{ConfigMap: *o}
		case *v1alpha1.Sync:
			sync := o
			// We only support one version at the moment.
			for _, group := range sync.Spec.Groups {
				for _, kind := range group.Kinds {
					if len(kind.Versions) > 1 {
						return nil, errors.Errorf("Sync declaration %s in file %s contains multiple "+
							"versions. Syncs must declare exactly one version.", sync.Name, i.Source)
					}
				}
			}
			// Disallow Sync for namespace type.
			for _, sg := range sync.Spec.Groups {
				for _, k := range sg.Kinds {
					if k.Kind == "Namespace" && sg.Group == "" {
						return nil, errors.Errorf("unsupported Sync in %s. Sync must not specify kind Namespace", i.Source)
					}
				}
			}
			syncs = append(syncs, sync)
		default:
			v.ObjectDisallowedInContext(i, o.GetObjectKind().GroupVersionKind())
		}
		if v.err != nil {
			return nil, v.err
		}
	}

	if fsCtx.Repo == nil {
		return nil, validation.MissingRepoError{}
	}
	return syncs, nil
}

// processClusterRegistryDir looks at all files in <root>/clusterregistry and
// extracts Cluster and ClusterSelector objects out. dirname is the directory
// name relative to the root directory of the repository, and infos is the set
// of resource data that were read from the directory.
func (p *Parser) processClusterRegistryDir(dirname string, infos []*resource.Info) ([]clusterregistry.Cluster, []v1alpha1.ClusterSelector, error) {
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
	return crc, css, v.err
}

// allDirs returns absolute paths of all directories in root, in lexicographic (depth-first) order.
func allDirs(root string) ([]string, error) {
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
		return nil, errors.Wrapf(err, "failed to walk directory: %s", root)
	}
	return paths, nil
}

// validateDirNames validates that:
// 1. Directory name is not reserved by the system.
// 2. Directory name is a valid namespace name:
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
func validateDirNames(dirs []string) error {
	dirNames := make(map[string]bool)
	for _, d := range dirs {
		n := filepath.Base(d)
		if err := namespaceutil.IsReservedOrInvalidNamespace(n); err != nil {
			return errors.Wrapf(err, "invalid directory: %s", d)
		}
		dirNames[n] = true

	}
	return nil
}

func validateDuplicateNames(dirInfos map[string][]*resource.Info) error {
	for _, infos := range dirInfos {
		gvkNameMap := map[schema.GroupVersionKind]map[string]bool{}

		for d, i := range infos {
			gvk := i.Mapping.GroupVersionKind
			if gvkNameMap[gvk] == nil {
				gvkNameMap[gvk] = map[string]bool{i.Name: true}
			} else if _, exists := gvkNameMap[gvk][i.Name]; !exists {
				gvkNameMap[gvk][i.Name] = true
			} else {
				return errors.Errorf("detected duplicate object %q of type %#v in directory %q",
					i.Name, gvk, d)
			}
		}
	}
	return nil
}
