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
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/blang/semver"
	policyhierarchyv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	policyhierarchyv1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/backend"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/namespaceutil"
	policynodevalidator "github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	kubevalidation "k8s.io/kubernetes/pkg/kubectl/validation"
)

// Parser reads files on disk and builds Nomos CRDs.
type Parser struct {
	factory   cmdutil.Factory
	resources []*metav1.APIResourceList
	validate  bool
}

const (
	// Directory names with special meaning in Nomos

	// systemDir is the name of the directory containing Nomos system configuration files.
	systemDir = "system"
	// treeDir is the name of the directory containin hierarchical resource
	// policies.
	treeDir = "tree"
	// clusterDir is the name of the directory where cluster scoped resources
	// are stored.
	clusterDir = "cluster"
	// clusterregistryDir is the relative path name to the directory containing
	// cluster registry information and cluster selectors.
	clusterregistryDir = "clusterregistry"
)

// NewParser creates a new Parser.
// clientConfig can be used to configure api server client. It should be set to nil when running in cluster.
// resources is the list returned by the DisoveryClient ServerResources call which represents resources
// 		that are returned by the API server during discovery.
// validate determines whether to validate schema using OpenAPI spec.
func NewParser(config clientcmd.ClientConfig, resources []*metav1.APIResourceList, validate bool) (*Parser, error) {
	p := Parser{
		factory:   cmdutil.NewFactory(config),
		resources: resources,
		validate:  validate,
	}

	return &p, nil
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// * system/ (may be absent)
// * cluster/
// * clusterregistry/ (may be absent)
// * tree/ (recursively)
func (p Parser) Parse(root string) (*policyhierarchyv1.AllPolicies, error) {
	fsCtx := &ast.Context{Cluster: &ast.Cluster{}}

	// Special processing for <root>/system/*
	var allowedGVKs map[schema.GroupVersionKind]struct{}
	if _, err := os.Stat(filepath.Join(root, systemDir)); err == nil {
		if allowedGVKs, err = p.processSystemDir(root, fsCtx); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		// IsNotExist is ok. We just won't read the directory.
		return nil, errors.Wrapf(err, "while checking existence of system directory")
	}

	// Special processing for <root>/cluster/*
	clusterDir := filepath.Join(root, clusterDir)
	clusterInfos, err := p.makeFileInfos(clusterDir, false)
	if err != nil {
		return nil, errors.Wrapf(err, "while making FileInfos for cluster")
	}

	// Special processing for <root>/clusterregistry/*
	var clusterregistryInfos []*resource.Info
	clusterregistryPath := filepath.Join(root, clusterregistryDir)
	if _, err = os.Stat(clusterregistryPath); err == nil {
		clusterregistryInfos, err = p.makeFileInfos(clusterregistryPath, false)
		if err != nil {
			return nil, errors.Wrapf(err, "while making FileInfos for clusterregistry")
		}
	} else if !os.IsNotExist(err) {
		// It's OK not to define the clusterregistry directory, you just won't get
		// the PCA features.
		return nil, errors.Wrapf(err, "while checking existence of clusterregistry directory")
	}

	// Regular recursive processing for <root>/tree/**/*
	treeDir := filepath.Join(root, treeDir)
	treeDirsOrdered, err := allDirs(treeDir)
	if err != nil {
		return nil, err
	}
	if err = validateDirNames(treeDirsOrdered); err != nil {
		return nil, err
	}
	fileInfos, err := p.makeFileInfos(treeDir, true)
	if err != nil {
		return nil, errors.Wrapf(err, "while making FileInfos for tree")
	}

	// TODO(filmil): dirInfos could just be map[string]runtime.Object, it seems.  Let's wait
	// until the new repo format commit lands, and change it then.
	dirInfos := make(map[string][]*resource.Info)
	// Value of all dirs is initially set to nil.
	for _, d := range treeDirsOrdered {
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

	return processDirs(p.resources, dirInfos, clusterInfos, clusterregistryInfos, treeDirsOrdered,
		clusterDir, fsCtx, allowedGVKs)
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

	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk directory: %s", root)
	}
	return paths, nil
}

// validateDirNames validates that:
// 1. No two directories (including root) in the tree have the same name.
// 2. Directory name is not reserved by the system.
// 3. Directory name is a valid namespace name:
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
func validateDirNames(dirs []string) error {
	dirNames := make(map[string]bool)
	for _, d := range dirs {
		n := filepath.Base(d)
		if _, ok := dirNames[n]; ok {
			return errors.Errorf("directory name %q is not unique: %s", n, d)
		}
		if err := namespaceutil.IsReservedOrInvalidNamespace(n); err != nil {
			return errors.Wrapf(err, "invalid directory: %s", d)
		}
		dirNames[n] = true

	}
	return nil
}

// makeFileInfos walks dir recursively, looking for resources, and builds FileInfos from them.
func (p Parser) makeFileInfos(dir string, recursive bool) ([]*resource.Info, error) {
	// If there aren't any resources, skip builder, because builder treats that as an error.
	var fileInfos []*resource.Info
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fileInfos, nil
	}
	visitors, err := resource.ExpandPathsToFileVisitors(
		&resource.Mapper{}, dir, true, resource.FileExtensions, kubevalidation.NullSchema{})
	if err != nil {
		return nil, err
	}
	if len(visitors) > 0 {
		schema, err := p.factory.Validator(p.validate)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get schema")
		}
		result := p.factory.NewBuilder().
			Unstructured().
			Schema(schema).
			ContinueOnError().
			FilenameParam(false, &resource.FilenameOptions{Recursive: recursive, Filenames: []string{dir}}).
			Do()
		fileInfos, err = result.Infos()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read resources from %s", dir)
		}
	}
	return fileInfos, nil
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

// processDirs validates objects in directory trees and converts them into hierarchical policy objects.
//
// clusterregistryInfos is the set of resources found in the directory <root>/clusterregistry.
//
// It first processes the cluster directory and then the tree hierarchy.
// cluster is a single, flat directory containing cluster-scoped resources.
// tree is hierarchical, containing 2 categories of directories:
// 1. Policyspace directory: Non-leaf directories at any depth within root directory.
// 2. Namespace directory: Leaf directories at any depth within root directory.
func processDirs(resources []*metav1.APIResourceList,
	dirInfos map[string][]*resource.Info,
	clusterInfos []*resource.Info,
	clusterregistryInfos []*resource.Info,
	treeDirsOrdered []string,
	clusterDir string,
	fsCtx *ast.Context,
	allowedGVKs map[schema.GroupVersionKind]struct{}) (*policyhierarchyv1.AllPolicies, error) {
	namespaceDirs := make(map[string]bool)

	treeGenerator := NewDirectoryTree()
	if err := processClusterDir(clusterDir, clusterInfos, fsCtx); err != nil {
		return nil, errors.Wrapf(err, "cluster directory is invalid: %s", clusterDir)
	}
	// TODO(filmil): Tie the processed results into a visitor.
	if _, _, err := processClusterRegistryDir(clusterregistryDir, clusterregistryInfos); err != nil {
		return nil, errors.Wrapf(err, "clusterregistry directory is invalid: %s", clusterDir)
	}

	if len(treeDirsOrdered) > 0 {
		rootDir := treeDirsOrdered[0]
		infos := dirInfos[rootDir]
		if err := processTreeDir(rootDir, infos, namespaceDirs, treeGenerator, true); err != nil {
			return nil, errors.Wrapf(err, "directory is invalid: %s", rootDir)
		}
		for _, d := range treeDirsOrdered[1:] {
			infos := dirInfos[d]
			if err := processTreeDir(d, infos, namespaceDirs, treeGenerator, false); err != nil {
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
		validation.NewInputValidator(allowedGVKs),
		scopeValidator,
		transform.NewAnnotationInlinerVisitor(),
		transform.NewInheritanceVisitor(
			[]transform.InheritanceSpec{
				{
					GroupVersionKind: rbacv1.SchemeGroupVersion.WithKind("RoleBinding"),
				},
			},
		),
		transform.NewQuotaVisitor(),
	}

	for _, visitor := range visitors {
		fsCtx = fsCtx.Accept(visitor).(*ast.Context)
		if err := visitor.Error(); err != nil {
			return nil, err
		}
	}

	outputVisitor := backend.NewOutputVisitor()
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

// applyPathAnnotation applies path annotation to o.
// dir is a slash-separated path.
func applyPathAnnotation(o runtime.Object, info *resource.Info, dir string) {
	metaObj := o.(metav1.Object)
	a := metaObj.GetAnnotations()
	if a == nil {
		a = map[string]string{}
		metaObj.SetAnnotations(a)
	}
	a[policyhierarchyv1alpha1.DeclarationPathAnnotationKey] = path.Join(dir, filepath.Base(info.Source))
}

func processClusterDir(
	dir string,
	infos []*resource.Info,
	fsCtx *ast.Context) error {
	path := path.Base(filepath.ToSlash(dir))
	for _, i := range infos {
		o := i.AsVersioned()
		applyPathAnnotation(o, i, path)
		fsCtx.Cluster.Objects = append(fsCtx.Cluster.Objects, &ast.ClusterObject{Object: o})
	}

	return nil
}

func processTreeDir(
	dir string,
	infos []*resource.Info,
	namespaceDirs map[string]bool,
	treeGenerator *DirectoryTree,
	root bool) error {
	parent := filepath.Dir(dir)
	// Since directories are processed in DFS order, it's guaranteed that parent was already processed.
	if namespaceDirs[parent] {
		return errors.Errorf("Namespace dir %s must not have children", parent)
	}
	var treeNode *ast.TreeNode
	for _, i := range infos {
		o := i.AsVersioned()

		switch o.(type) {
		case *corev1.Namespace:
			namespaceDirs[dir] = true
			if root {
				treeNode = treeGenerator.SetRootDir(dir, ast.Namespace)
			} else {
				treeNode = treeGenerator.AddDir(dir, ast.Namespace)
			}
			return processNamespaceDir(dir, infos, treeNode)
		}
	}
	// No namespace resource was found.
	if root {
		treeNode = treeGenerator.SetRootDir(dir, ast.Policyspace)
	} else {
		treeNode = treeGenerator.AddDir(dir, ast.Policyspace)
	}
	return processPolicyspaceDir(infos, treeNode)
}

func processPolicyspaceDir(infos []*resource.Info, treeNode *ast.TreeNode) error {
	for _, i := range infos {
		o := i.AsVersioned()
		applyPathAnnotation(o, i, treeNode.Path)

		switch o.GetObjectKind().GroupVersionKind() {
		case policyhierarchyv1alpha1.SchemeGroupVersion.WithKind("NamespaceSelector"):
			if err := parseNamespaceSelector(i.Object, treeNode, i.Source); err != nil {
				return err
			}
		default:
			treeNode.Objects = append(treeNode.Objects, &ast.Object{Object: o})
		}
	}

	return nil
}

func processNamespaceDir(dir string, infos []*resource.Info, treeNode *ast.TreeNode) error {
	namespace := filepath.Base(dir)
	v := newValidator()

	for _, i := range infos {
		o := i.AsVersioned()
		applyPathAnnotation(o, i, treeNode.Path)

		gvk := o.GetObjectKind().GroupVersionKind()
		if gvk == corev1.SchemeGroupVersion.WithKind("Namespace") {
			// TODO: Move this out.
			metaObj := o.(metav1.Object)
			treeNode.Labels = metaObj.GetLabels()
			treeNode.Annotations = metaObj.GetAnnotations()
			v.HasName(i, namespace).HaveNotSeen(gvk).MarkSeen(gvk)
			continue
		}

		if o.GetObjectKind().GroupVersionKind() == policyhierarchyv1alpha1.SchemeGroupVersion.WithKind("NamespaceSelector") {
			v.ObjectDisallowedInContext(i, o.GetObjectKind().GroupVersionKind())
		}
		if v.err != nil {
			return v.err
		}

		treeNode.Objects = append(treeNode.Objects, &ast.Object{Object: o})
	}

	v.HaveSeen(schema.GroupVersionKind{Version: "v1", Kind: "Namespace"})
	if v.err != nil {
		return v.err
	}

	return nil
}

// parseNamespaceSelector converts adds a NamespaceSelector into the provided
// node.  o must be known to be a NamespaceSelector.  source is the source file
// the object was read from, for error diagnostics only and may be set to "" if
// unknown.
func parseNamespaceSelector(o runtime.Object, node *ast.TreeNode, source string) error {
	ns := &policyhierarchyv1alpha1.NamespaceSelector{}
	if err := convertUnstructured(o, ns, source); err != nil {
		return err
	}

	if node.Selectors == nil {
		node.Selectors = make(map[string]*policyhierarchyv1alpha1.NamespaceSelector)
	}
	node.Selectors[ns.Name] = ns
	return nil
}

// convertUnstructured converts a runtime.Unstructured to a specifc type.  The hope is that we can
// eventually replace the call to json.Marshal/Unmarshal with some form of officially supported
// APIMachinery code.
func convertUnstructured(o runtime.Object, want interface{}, source string) error {
	wantKind := reflect.TypeOf(want).Elem().Kind().String()
	j, err := json.Marshal(o)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal object in %s to %s", source, wantKind)
	}
	err = json.Unmarshal(j, want)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal %s", wantKind)
	}
	return nil
}

// processSystemDir loads configs from <root>/system/nomos.yaml.
func (p Parser) processSystemDir(root string, fsCtx *ast.Context) (map[schema.GroupVersionKind]struct{}, error) {
	validator, err := p.factory.Validator(p.validate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get schema")
	}
	result := p.factory.NewBuilder().
		Unstructured().
		Schema(validator).
		ContinueOnError().
		FilenameParam(false, &resource.FilenameOptions{Recursive: false, Filenames: []string{filepath.Join(root, systemDir)}}).
		Do()
	fileInfos, err := result.Infos()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read resources from system dir")
	}
	v := newValidator()
	allowedGVKs := make(map[schema.GroupVersionKind]struct{})
	for _, i := range fileInfos {
		o := i.AsVersioned()

		// Types in scope then alphabetical order.
		switch gvk := o.GetObjectKind().GroupVersionKind(); gvk {
		case policyhierarchyv1alpha1.SchemeGroupVersion.WithKind("NomosConfig"):
			fsCtx.Config, err = parseNomosConfig(o, i.Source)
			if err != nil {
				v.err = errors.Wrapf(err, "failed to parse NomosConfig in %s", i.Source)
			}
		case corev1.SchemeGroupVersion.WithKind("ConfigMap"):
			cm := o.(*corev1.ConfigMap)
			if cm.Name == policyhierarchyv1alpha1.ReservedNamespacesConfigMapName {
				fsCtx.ReservedNamespaces = &ast.ReservedNamespaces{ConfigMap: *cm}
			} else {
				v.err = errors.Errorf("Invalid configmap in system dir %#v", o)
			}
		case policyhierarchyv1alpha1.SchemeGroupVersion.WithKind("Sync"):
			var sync *policyhierarchyv1alpha1.Sync
			if sync, err = parseSync(i.Object, i.Source); err != nil {
				return nil, errors.Wrapf(err, "failed to parse Sync in %s", i.Source)
			}
			for _, sg := range sync.Spec.Groups {
				for _, k := range sg.Kinds {
					if k.Kind == "Namespace" && sg.Group == "" {
						return nil, errors.Errorf("unsupported Sync in %s. Sync must not specify kind Namespace", i.Source)
					}
					for _, v := range k.Versions {
						gvk := schema.GroupVersionKind{Group: sg.Group, Kind: k.Kind, Version: v.Version}
						allowedGVKs[gvk] = struct{}{}
					}
				}
			}
		default:
			v.ObjectDisallowedInContext(i, o.GetObjectKind().GroupVersionKind())
		}
		if v.err != nil {
			return nil, v.err
		}
	}

	if fsCtx.Config == nil {
		return nil, errors.Errorf("failed to find object of type NomosConfig in system dir")
	}
	return allowedGVKs, nil
}

// parseNomosConfig parses out a NomosConfig object from o, which must be a NomosConfig object.
// source is the source file the object was read from.
func parseNomosConfig(o runtime.Object, source string) (*policyhierarchyv1alpha1.NomosConfig, error) {
	config := &policyhierarchyv1alpha1.NomosConfig{}
	if err := convertUnstructured(o, config, source); err != nil {
		return nil, err
	}

	if _, err := semver.Parse(config.Spec.RepoVersion); err != nil {
		return nil, errors.Wrapf(err, "invalid semantic version %s. "+
			"NomosConfig.Spec.RepoVersion must follow semantic versioning rules at http://semver.org",
			config.Spec.RepoVersion)
	}

	return config, nil
}

// parseSync parses a policyhierarchyv1alpha1.Sync from o, which must be a Sync object.
// source is the source file the object was read from.
func parseSync(o runtime.Object, source string) (*policyhierarchyv1alpha1.Sync, error) {
	sync := &policyhierarchyv1alpha1.Sync{}
	if err := convertUnstructured(o, sync, source); err != nil {
		return nil, err
	}
	for _, group := range sync.Spec.Groups {
		for _, kind := range group.Kinds {
			if len(kind.Versions) > 1 {
				return nil, errors.Errorf("Sync declaration %s in file %s contains multiple "+
					"versions. Syncs must declare exactly one version.", o.(metav1.Object).GetName(), source)
			}
		}
	}
	return sync, nil
}
