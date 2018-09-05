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
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/backend"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform"
	"github.com/google/nomos/pkg/util/clusterpolicy"
	"github.com/google/nomos/pkg/util/namespaceutil"
	policynodevalidator "github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	extensions_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

// Parser reads files on disk and builds Nomos CRDs.
type Parser struct {
	factory  cmdutil.Factory
	validate bool
}

// NewParser creates a new Parser.
// clientConfig can be used to configure api server client. It should be set to nil when running in cluster.
// validate determines whether to validate schema using OpenAPI spec.
func NewParser(config clientcmd.ClientConfig, validate bool) (*Parser, error) {
	p := Parser{
		factory:  cmdutil.NewFactory(config),
		validate: validate,
	}

	return &p, nil
}

// Parse recursively parses file tree rooted at root and builds policy CRDs from
// supported Kubernetes policy resources.
func (p Parser) Parse(root string) (*policyhierarchy_v1.AllPolicies, error) {
	allDirsOrdered, err := allDirs(root)
	if err != nil {
		return nil, err
	}
	if err = validateDirNames(allDirsOrdered); err != nil {
		return nil, err
	}

	// Walk the filesystem looking for resources. If there aren't any, skip builder, because builder
	// treats that as an error.
	visitors, err := resource.ExpandPathsToFileVisitors(
		&resource.Mapper{}, root, true, resource.FileExtensions, validation.NullSchema{})
	if err != nil {
		return nil, err
	}
	var fileInfos []*resource.Info
	if len(visitors) > 0 {
		schema, err := p.factory.Validator(p.validate)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get schema")
		}
		result := p.factory.NewBuilder().
			Unstructured().
			Schema(schema).
			ContinueOnError().
			FilenameParam(false, &resource.FilenameOptions{Recursive: true, Filenames: []string{root}}).
			Do()
		fileInfos, err = result.Infos()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read resources from %s", root)
		}
	}

	dirInfos := make(map[string][]*resource.Info)
	// Value of all dirs is initially set to nil.
	for _, d := range allDirsOrdered {
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

	return processDirs(dirInfos, allDirsOrdered)
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

// processDirs validates objects in directory tree and converts them into hierarchical policy objects.
// There are 3 categories of directories:
// 1. Root directory: Top-level directory given by root.
// 2. Policyspace directory: Non-leaf directories at any depth within root directory.
// 3. Namespace directory: Leaf directories at any depth within root directory.
func processDirs(dirInfos map[string][]*resource.Info, allDirsOrdered []string) (*policyhierarchy_v1.AllPolicies, error) {
	namespaceDirs := make(map[string]bool)

	root := allDirsOrdered[0]
	rootInfos := dirInfos[root]

	treeGenerator := NewDirectoryTree()
	fsCtx, err := processRootDir(root, rootInfos, treeGenerator)
	if err != nil {
		return nil, errors.Wrapf(err, "root directory is invalid: %s", root)
	}

	for _, d := range allDirsOrdered[1:] {
		infos := dirInfos[d]
		err2 := processNonRootDir(d, infos, namespaceDirs, treeGenerator)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "directory is invalid: %s", d)
		}
	}

	tree, err := treeGenerator.Build()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to treeify policy nodes")
	}
	fsCtx.Tree = tree

	visitors := []ast.MutatingVisitor{
		NewInputValidator(),
		transform.NewInheritanceVisitor(
			[]transform.InheritanceSpec{
				{
					GroupVersionKind:  rbac_v1.SchemeGroupVersion.WithKind("RoleBinding"),
					PolicyspacePrefix: true,
				},
			},
		),
		transform.NewQuotaVisitor(),
	}

	for _, visitor := range visitors {
		fsCtx = fsCtx.Accept(visitor).(*ast.Context)
		if err := visitor.Result(); err != nil {
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

func applyPathAnnotation(o runtime.Object, info *resource.Info, dir string) {
	metaObj := o.(meta_v1.Object)
	a := metaObj.GetAnnotations()
	if a == nil {
		a = map[string]string{}
		metaObj.SetAnnotations(a)
	}
	a[policyhierarchy_v1.AnnotationKeyDeclarationPath] = filepath.Join(dir, filepath.Base(info.Source))
}

func processRootDir(
	dir string,
	infos []*resource.Info,
	treeGenerator *DirectoryTree) (*ast.Context, error) {
	v := newValidator()
	fsCtx := &ast.Context{Cluster: &ast.Cluster{}}
	rootNode := treeGenerator.SetRootDir(dir)

	for _, i := range infos {
		o := i.AsVersioned()
		applyPathAnnotation(o, i, rootNode.Path)

		// Types in scope then alphabetical order.
		switch o := o.(type) {
		// System scope
		case *core_v1.ConfigMap:
			if o.Name == policyhierarchy_v1.ReservedNamespacesConfigMapName {
				fsCtx.ReservedNamespaces = &ast.ReservedNamespaces{ConfigMap: *o}
			} else {
				v.err = errors.Errorf("Invalid configmap in root dir %#v", o)
			}

			// Cluster scope
		case *rbac_v1.ClusterRole:
			fsCtx.Cluster.Objects = append(fsCtx.Cluster.Objects, &ast.Object{Object: o})
		case *rbac_v1.ClusterRoleBinding:
			fsCtx.Cluster.Objects = append(fsCtx.Cluster.Objects, &ast.Object{Object: o})
		case *core_v1.Namespace:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *extensions_v1beta1.PodSecurityPolicy:
			fsCtx.Cluster.Objects = append(fsCtx.Cluster.Objects, &ast.Object{Object: o})

			// Namespace Scope
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, "")
			rootNode.Objects = append(rootNode.Objects, &ast.Object{Object: o})
		case *rbac_v1.Role:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, "")
			rootNode.Objects = append(rootNode.Objects, &ast.Object{Object: o})
		case runtime.Unstructured:
			switch o.GetObjectKind().GroupVersionKind() {
			case policyhierarchy_v1.SchemeGroupVersion.WithKind("NamespaceSelector"):
				v.err = parseNamespaceSelector(o, rootNode)
			default:
				glog.Warningf("Ignoring unsupported unstructured object %q in %s", o.GetObjectKind().GroupVersionKind(), i.Source)
			}
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
		}

		if v.err != nil {
			return nil, v.err
		}
	}

	return fsCtx, nil
}

func processNonRootDir(
	dir string,
	infos []*resource.Info,
	namespaceDirs map[string]bool,
	treeGenerator *DirectoryTree) error {
	parent := filepath.Dir(dir)
	// Since directories are processed in DFS order, it's guaranteed that parent was already processed.
	if namespaceDirs[parent] {
		return errors.Errorf("Namespace dir %s must not have children", parent)
	}
	for _, i := range infos {
		o := i.AsVersioned()

		switch o.(type) {
		case *core_v1.Namespace:
			namespaceDirs[dir] = true
			return processNamespaceDir(dir, infos, treeGenerator)
		}
	}
	// No namespace resource was found.
	return processPolicyspaceDir(dir, infos, treeGenerator)
}

func processPolicyspaceDir(dir string, infos []*resource.Info, treeGenerator *DirectoryTree) error {
	v := newValidator()
	treeNode := treeGenerator.AddDir(dir, ast.Policyspace)

	for _, i := range infos {
		o := i.AsVersioned()
		applyPathAnnotation(o, i, treeNode.Path)

		// Types in alphabetical order.
		switch o := o.(type) {
		case *rbac_v1.ClusterRole:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *rbac_v1.ClusterRoleBinding:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *core_v1.Namespace:
			// Should not be reachable.
			panic(fmt.Sprintf("%s processed as policyspace but contains a namespace resource", dir))
		case *extensions_v1beta1.PodSecurityPolicy:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, "")
			treeNode.Objects = append(treeNode.Objects, &ast.Object{Object: o})
		case *rbac_v1.Role:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, "")
			treeNode.Objects = append(treeNode.Objects, &ast.Object{Object: o})
		case *core_v1.ConfigMap:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case runtime.Unstructured:
			switch o.GetObjectKind().GroupVersionKind() {
			case policyhierarchy_v1.SchemeGroupVersion.WithKind("NamespaceSelector"):
				v.err = parseNamespaceSelector(o, treeNode)
			default:
				glog.Warningf("Ignoring unsupported unstructured object %q in %s", o.GetObjectKind().GroupVersionKind(), i.Source)
			}
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
		}

		if v.err != nil {
			return v.err
		}
	}

	return nil
}

func processNamespaceDir(dir string, infos []*resource.Info, treeGenerator *DirectoryTree) error {
	namespace := filepath.Base(dir)
	v := newValidator()
	treeNode := treeGenerator.AddDir(dir, ast.Namespace)

	for _, i := range infos {
		o := i.AsVersioned()
		applyPathAnnotation(o, i, treeNode.Path)

		// Types in alphabetical order.
		switch o := o.(type) {
		case *rbac_v1.ClusterRole:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *rbac_v1.ClusterRoleBinding:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *core_v1.Namespace:
			v.HasName(i, namespace).HaveNotSeen(o.GroupVersionKind()).MarkSeen(o.GroupVersionKind())
			treeNode.Labels = o.Labels
			treeNode.Annotations = o.Annotations
		case *extensions_v1beta1.PodSecurityPolicy:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, namespace).HaveNotSeen(o.GroupVersionKind()).MarkSeen(o.GroupVersionKind())
			treeNode.Objects = append(treeNode.Objects, &ast.Object{Object: o})
		case *rbac_v1.Role:
			v.HasNamespace(i, namespace)
			treeNode.Objects = append(treeNode.Objects, &ast.Object{Object: o})
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, namespace)
			treeNode.Objects = append(treeNode.Objects, &ast.Object{Object: o})
		case *core_v1.ConfigMap:
			v.ObjectDisallowedInContext(i, o.GroupVersionKind())
		case runtime.Unstructured:
			switch o.GetObjectKind().GroupVersionKind() {
			case policyhierarchy_v1.SchemeGroupVersion.WithKind("NamespaceSelector"):
				v.ObjectDisallowedInContext(i, o.GetObjectKind().GroupVersionKind())
			default:
				glog.Warningf("Ignoring unsupported unstructured object %q in %s", o.GetObjectKind().GroupVersionKind(), i.Source)
			}
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
			continue
		}

		if v.err != nil {
			return v.err
		}
	}

	v.HaveSeen(schema.GroupVersionKind{Version: "v1", Kind: "Namespace"})
	if v.err != nil {
		return v.err
	}

	return nil
}

func parseNamespaceSelector(o runtime.Unstructured, node *ast.TreeNode) error {
	j, err := json.Marshal(o.UnstructuredContent())
	if err != nil {
		return errors.Wrap(err, "failed to marshal object in %s to NamespaceSelector")
	}
	ns := &policyhierarchy_v1.NamespaceSelector{}
	err = json.Unmarshal(j, ns)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal NamespaceSelector")
	}

	if node.Selectors == nil {
		node.Selectors = make(map[string]*policyhierarchy_v1.NamespaceSelector)
	}
	node.Selectors[ns.Name] = ns
	return nil
}
