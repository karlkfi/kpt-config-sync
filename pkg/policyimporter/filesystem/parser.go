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
	"path/filepath"

	"fmt"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/util/namespaceutil"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	extensions_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

// Parser reads files on disk and builds Nomos CRDs.
type Parser struct {
	factory   cmdutil.Factory
	schema    validation.Schema
	inCluster bool
}

// NewParser creates a new Parser.
// inCluster boolean determines if this is running in a cluster and can talk to api server.
func NewParser(inCluster bool) (*Parser, error) {
	p := Parser{
		factory:   cmdutil.NewFactory(nil),
		inCluster: inCluster,
	}

	// If running in cluster, validate objects using OpenAPI schema downloaded from the API server.
	// TODO(frankfarzan): Bake in the swagger spec when running outside the cluster?
	if inCluster {
		schema, err := p.factory.Validator(true)
		if err != nil {
			return nil, errors.Wrap(err, "fail to get schema")
		}
		p.schema = schema
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
		builder := p.factory.NewBuilder().Internal()
		if p.inCluster {
			builder = builder.Schema(p.schema)
		} else {
			builder = builder.Local()
		}
		result := builder.
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

	var policies policyhierarchy_v1.AllPolicies
	policies.PolicyNodes = make(map[string]policyhierarchy_v1.PolicyNode)

	root := allDirsOrdered[0]
	rootInfos := dirInfos[root]
	p, c, err := processRootDir(root, rootInfos)
	if err != nil {
		return nil, errors.Wrapf(err, "root directory is invalid: %s", root)
	}
	policies.PolicyNodes[p.Name] = *p
	policies.ClusterPolicy = c

	for _, d := range allDirsOrdered[1:] {
		infos := dirInfos[d]
		p, err := processNonRootDir(d, infos, namespaceDirs)
		if err != nil {
			return nil, errors.Wrapf(err, "directory is invalid: %s", d)
		}
		policies.PolicyNodes[p.Name] = *p
	}

	return &policies, nil
}

func processRootDir(dir string, infos []*resource.Info) (*policyhierarchy_v1.PolicyNode, *policyhierarchy_v1.ClusterPolicy, error) {
	v := newValidator()
	pn := policynode.NewPolicyNode(
		filepath.Base(dir),
		&policyhierarchy_v1.PolicyNodeSpec{
			Type:   policyhierarchy_v1.Policyspace,
			Parent: policyhierarchy_v1.NoParentNamespace,
		},
	)
	cp := policynode.NewClusterPolicy(
		policyhierarchy_v1.ClusterPolicyName,
		&policyhierarchy_v1.ClusterPolicySpec{},
	)

	for _, i := range infos {
		o := i.AsVersioned()

		// Types in alphabetical order.
		switch o := o.(type) {
		case *rbac_v1.ClusterRole:
			cp.Spec.ClusterRolesV1 = append(cp.Spec.ClusterRolesV1, *o)
		case *rbac_v1.ClusterRoleBinding:
			cp.Spec.ClusterRoleBindingsV1 = append(cp.Spec.ClusterRoleBindingsV1, *o)
		case *core_v1.Namespace:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *extensions_v1beta1.PodSecurityPolicy:
			cp.Spec.PodSecurityPoliciesV1Beta1 = append(cp.Spec.PodSecurityPoliciesV1Beta1, *o)
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, "")
			pn.Spec.ResourceQuotaV1 = o
		case *rbac_v1.Role:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, "")
			pn.Spec.RoleBindingsV1 = append(pn.Spec.RoleBindingsV1, *o)
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
		}

		if v.err != nil {
			return nil, nil, v.err
		}
	}
	// There's a singleton ClusterPolicy object for the hierarchy.
	return pn, cp, nil
}

func processNonRootDir(dir string, infos []*resource.Info, namespaceDirs map[string]bool) (*policyhierarchy_v1.PolicyNode, error) {
	parent := filepath.Dir(dir)
	// Since directories are processed in DFS order, it's guaranteed that parent was already processed.
	if namespaceDirs[parent] {
		return nil, errors.Errorf("Namespace dir %s must not have children", parent)
	}
	for _, i := range infos {
		o := i.AsVersioned()

		switch o.(type) {
		case *core_v1.Namespace:
			namespaceDirs[dir] = true
			return processNamespaceDir(dir, infos)
		}
	}
	// No namespace resource was found.
	return processPolicyspaceDir(dir, infos)
}

func processPolicyspaceDir(dir string, infos []*resource.Info) (*policyhierarchy_v1.PolicyNode, error) {
	v := newValidator()
	pn := policynode.NewPolicyNode(
		filepath.Base(dir),
		&policyhierarchy_v1.PolicyNodeSpec{
			Type:   policyhierarchy_v1.Policyspace,
			Parent: filepath.Base(filepath.Dir(dir)),
		},
	)

	for _, i := range infos {
		o := i.AsVersioned()

		// Types in alphabetical order.
		switch o := o.(type) {
		case *rbac_v1.ClusterRole:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *rbac_v1.ClusterRoleBinding:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *core_v1.Namespace:
			// Should not be reachable.
			panic(fmt.Sprintf("%s processed as policyspace but contains a namespace resource", dir))
		case *extensions_v1beta1.PodSecurityPolicy:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, "")
			pn.Spec.ResourceQuotaV1 = o
		case *rbac_v1.Role:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, "")
			pn.Spec.RoleBindingsV1 = append(pn.Spec.RoleBindingsV1, *o)
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
		}

		if v.err != nil {
			return nil, v.err
		}
	}

	return pn, nil
}

func processNamespaceDir(dir string, infos []*resource.Info) (*policyhierarchy_v1.PolicyNode, error) {
	namespace := filepath.Base(dir)
	v := newValidator()
	pn := policynode.NewPolicyNode(
		filepath.Base(dir),
		&policyhierarchy_v1.PolicyNodeSpec{
			Type:   policyhierarchy_v1.Namespace,
			Parent: filepath.Base(filepath.Dir(dir)),
		},
	)

	for _, i := range infos {
		o := i.AsVersioned()

		// Types in alphabetical order.
		switch o := o.(type) {
		case *rbac_v1.ClusterRole:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *rbac_v1.ClusterRoleBinding:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *core_v1.Namespace:
			v.HasName(i, namespace).HaveNotSeen(o.TypeMeta).MarkSeen(o.TypeMeta)
		case *extensions_v1beta1.PodSecurityPolicy:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, namespace).HaveNotSeen(o.TypeMeta).MarkSeen(o.TypeMeta)
			pn.Spec.ResourceQuotaV1 = o
		case *rbac_v1.Role:
			v.HasNamespace(i, namespace)
			pn.Spec.RolesV1 = append(pn.Spec.RolesV1, *o)
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, namespace)
			pn.Spec.RoleBindingsV1 = append(pn.Spec.RoleBindingsV1, *o)
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
			continue
		}

		if v.err != nil {
			return nil, v.err
		}
	}

	v.HaveSeen(meta_v1.TypeMeta{Kind: "Namespace", APIVersion: "v1"})
	if v.err != nil {
		return nil, v.err
	}

	return pn, nil
}
