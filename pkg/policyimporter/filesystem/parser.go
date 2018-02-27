/*
Copyright 2017 The Stolos Authors.
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
// and converting them to Stolos Custom Resource Definition objects.
package filesystem

import (
	"os"
	"path/filepath"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/util/namespaceutil"
	"github.com/google/stolos/pkg/util/policynode"
	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
	extensions_v1beta1 "k8s.io/api/extensions/v1beta1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/kubectl/validation"
)

type parser struct {
	factory   cmdutil.Factory
	schema    validation.Schema
	inCluster bool
}

// NewParser creates a new parser that reads files on disk and builds Stolos CRDs.
// inCluster boolean determines if this is running in a cluster and can talk to api server.
func NewParser(inCluster bool) (*parser, error) {
	p := parser{
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
func (p parser) Parse(root string) (*policyhierarchy_v1.AllPolicies, error) {
	allDirs, err := allDirs(root)
	if err != nil {
		return nil, err
	}
	if err := validateDirNames(allDirs); err != nil {
		return nil, err
	}

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
	infos, err := result.Infos()
	if err != nil {
		// TODO(frankfarzan): error message does not contain the source of the error for some reason which is bad UX.
		// kubectl apply -R -f" has the same issue.
		// TODO(frankfarzan): error message contains flags specific to kubectl cmd.
		return nil, errors.Wrapf(err, "failed to read resources from %s", root)
	}

	dirInfos := make(map[string][]*resource.Info)
	// Value of all dirs is initially set to nil.
	for _, d := range allDirs {
		dirInfos[d] = nil
	}
	// If a directory has resources, its value in the map
	// will be non-nil.
	for _, i := range infos {
		d := filepath.Dir(i.Source)
		dirInfos[d] = append(dirInfos[d], i)
	}

	return processDirs(root, dirInfos)
}

// allDirs returns absolute paths of all directories in root.
// First element is root itself.
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
			return errors.Errorf("directory name %s is not unique in root %s", n, dirs[0])
		}
		if err := namespaceutil.IsReservedOrInvalidNamespace(n); err != nil {
			return errors.Wrapf(err, "invalid directory: %s", d)
		}
		dirNames[n] = true

	}
	return nil
}

// processDirs validates objects in directory tree and converts them into hierarchical policy objects.
// There are 3 categories of directories:
// 1. Root directory: Top-level directory given by root.
// 2. Policyspace directory: Non-leaf directories at any depth within root directory.
// 3. Namespace directory: Leaf directories at any depth within root directory.
func processDirs(root string, dirInfos map[string][]*resource.Info) (*policyhierarchy_v1.AllPolicies, error) {
	policyspaceDirs := make(map[string]bool)
	// Determine if dir is a policyspace by checking if it's a parent of any other dir.
	for d := range dirInfos {
		policyspaceDirs[filepath.Dir(d)] = true
	}

	var policies policyhierarchy_v1.AllPolicies
	policies.PolicyNodes = make(map[string]policyhierarchy_v1.PolicyNode)

	for d, infos := range dirInfos {
		if d == root {
			p, c, err := processRootDir(d, infos)
			if err != nil {
				return nil, errors.Wrapf(err, "root directory is invalid: %s", d)
			}
			policies.PolicyNodes[p.Name] = *p
			policies.ClusterPolicy = c
		} else if policyspaceDirs[d] {
			p, err := processPolicyspaceDir(d, infos)
			if err != nil {
				return nil, errors.Wrapf(err, "policyspace directory is invalid: %s", d)
			}
			policies.PolicyNodes[p.Name] = *p
		} else {
			p, err := processNamespaceDir(d, infos)
			if err != nil {
				return nil, errors.Wrapf(err, "namespace directory is invalid: %s", d)
			}
			policies.PolicyNodes[p.Name] = *p
		}
	}

	return &policies, nil
}

func processRootDir(dir string, infos []*resource.Info) (*policyhierarchy_v1.PolicyNode, *policyhierarchy_v1.ClusterPolicy, error) {
	var policies policyhierarchy_v1.Policies
	var clusterPolicies policyhierarchy_v1.ClusterPolicies
	v := newValidator()

	for _, i := range infos {
		o := i.AsVersioned()

		// Types in alphabetical order.
		switch o := o.(type) {
		case *rbac_v1.ClusterRole:
			clusterPolicies.ClusterRolesV1 = append(clusterPolicies.ClusterRolesV1, *o)
		case *rbac_v1.ClusterRoleBinding:
			clusterPolicies.ClusterRoleBindingsV1 = append(clusterPolicies.ClusterRoleBindingsV1, *o)
		case *core_v1.Namespace:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *extensions_v1beta1.PodSecurityPolicy:
			clusterPolicies.PodSecurtiyPoliciesV1Beta1 = append(clusterPolicies.PodSecurtiyPoliciesV1Beta1, *o)
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, "")
			policies.ResourceQuotaV1 = o.Spec
		case *rbac_v1.Role:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, "")
			policies.RoleBindingsV1 = append(policies.RoleBindingsV1, *o)
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
		}

		if v.err != nil {
			return nil, nil, v.err
		}
	}

	pn := policynode.NewPolicyNode(filepath.Base(dir),
		&policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: true,
			Parent:      policyhierarchy_v1.NoParentNamespace,
			Policies:    policies,
		})

	// There's a single ClusterPolicy object for the hierarchy.
	// Set its name to the root dir.
	cp := policynode.NewClusterPolicy(filepath.Base(dir),
		&policyhierarchy_v1.ClusterPolicySpec{
			Policies: clusterPolicies,
		})

	return pn, cp, nil
}

func processPolicyspaceDir(dir string, infos []*resource.Info) (*policyhierarchy_v1.PolicyNode, error) {
	var policies policyhierarchy_v1.Policies
	v := newValidator()

	for _, i := range infos {
		o := i.AsVersioned()

		// Types in alphabetical order.
		switch o := o.(type) {
		case *rbac_v1.ClusterRole:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *rbac_v1.ClusterRoleBinding:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *core_v1.Namespace:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *extensions_v1beta1.PodSecurityPolicy:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *core_v1.ResourceQuota:
			v.HasNamespace(i, "")
			policies.ResourceQuotaV1 = o.Spec
		case *rbac_v1.Role:
			v.ObjectDisallowedInContext(i, o.TypeMeta)
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, "")
			policies.RoleBindingsV1 = append(policies.RoleBindingsV1, *o)
		default:
			glog.Warningf("Ignoring unsupported object type %T in %s", o, i.Source)
		}

		if v.err != nil {
			return nil, v.err
		}
	}

	pn := policynode.NewPolicyNode(filepath.Base(dir),
		&policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: true,
			Parent:      filepath.Base(filepath.Dir(dir)),
			Policies:    policies,
		})

	return pn, nil
}

func processNamespaceDir(dir string, infos []*resource.Info) (*policyhierarchy_v1.PolicyNode, error) {
	namespace := filepath.Base(dir)
	var policies policyhierarchy_v1.Policies
	v := newValidator()

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
			// TODO(frankfarzan): policies.ResourceQuotaV1 = *o
			policies.ResourceQuotaV1 = o.Spec
		case *rbac_v1.Role:
			v.HasNamespace(i, namespace)
			policies.RolesV1 = append(policies.RolesV1, *o)
		case *rbac_v1.RoleBinding:
			v.HasNamespace(i, namespace)
			policies.RoleBindingsV1 = append(policies.RoleBindingsV1, *o)
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

	pn := policynode.NewPolicyNode(namespace,
		&policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: false,
			Parent:      filepath.Base(filepath.Dir(dir)),
			Policies:    policies,
		})

	return pn, nil
}
