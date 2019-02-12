/*
Copyright 2018 The Nomos Authors.

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

package testing

import (
	"time"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

var (
	// ImportToken defines a default token to use for testing.
	ImportToken = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	// ImportTime defines a default time to use for testing.
	ImportTime = time.Date(2017, 8, 10, 5, 16, 00, 0, time.FixedZone("PDT", -7*60*60))
)

// ObjectSets constructs a list of ObjectSet from a list of runtime.Object.
func ObjectSets(runtimeObjs ...runtime.Object) []*ast.NamespaceObject {
	astObjs := make([]*ast.NamespaceObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.NamespaceObject{FileObject: ast.FileObject{Object: runtimeObjs[idx]}}
	}
	return astObjs
}

// FileObjectSets constructs a list of ObjectSet from a list of ast.FileObject
func FileObjectSets(runtimeObjs ...ast.FileObject) []*ast.NamespaceObject {
	astObjs := make([]*ast.NamespaceObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.NamespaceObject{FileObject: runtimeObjs[idx]}
	}
	return astObjs
}

// ClusterObjectSets constructs a list of ObjectSet from a list of runtime.NamespaceObject.
func ClusterObjectSets(runtimeObjs ...runtime.Object) []*ast.ClusterObject {
	astObjs := make([]*ast.ClusterObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.ClusterObject{FileObject: ast.FileObject{Object: runtimeObjs[idx]}}
	}
	return astObjs
}

// ClusterRegistryObjectSets constructs a list of ObjectSet from a list of runtime.NamespaceObject.
func ClusterRegistryObjectSets(runtimeObjs ...runtime.Object) []*ast.ClusterRegistryObject {
	astObjs := make([]*ast.ClusterRegistryObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.ClusterRegistryObject{FileObject: ast.FileObject{Object: runtimeObjs[idx]}}
	}
	return astObjs
}

// SystemObjectSets constructs a list of ObjectSet from a list of runtime.NamespaceObject.
func SystemObjectSets(runtimeObjs ...runtime.Object) []*ast.SystemObject {
	astObjs := make([]*ast.SystemObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.SystemObject{FileObject: ast.FileObject{Object: runtimeObjs[idx]}}
	}
	return astObjs
}

// Helper provides a number of pre-built types for use in testcases.  This does not set an ImportToken
// or ImportTime
var Helper TestHelper

// TestHelper provides a number of pre-built types for use in testcases.
type TestHelper struct {
	ImportToken string
	ImportTime  time.Time
}

// NewTestHelper returns a TestHelper with default import token and time.
func NewTestHelper() *TestHelper {
	return &TestHelper{
		ImportToken: ImportToken,
		ImportTime:  ImportTime,
	}
}

// NomosAdminClusterRole returns a ClusterRole for testing.
func (t *TestHelper) NomosAdminClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nomos:admin",
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{rbacv1.VerbAll},
			APIGroups: []string{"nomos.dev"},
			Resources: []string{rbacv1.ResourceAll},
		}},
	}
}

// NomosAdminClusterRoleBinding returns a ClusterRoleBinding for testing.
func (t *TestHelper) NomosAdminClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "nomos:admin",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     "charlie@acme.com",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "nomos:admin",
		},
	}
}

// NomosPodSecurityPolicy returns a PodSecurityPolicy for testing.
func (t *TestHelper) NomosPodSecurityPolicy() *extensionsv1beta1.PodSecurityPolicy {
	return &extensionsv1beta1.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: extensionsv1beta1.SchemeGroupVersion.String(),
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "example",
		},
		Spec: extensionsv1beta1.PodSecurityPolicySpec{
			Privileged: false,
			RunAsUser: extensionsv1beta1.RunAsUserStrategyOptions{
				Rule: extensionsv1beta1.RunAsUserStrategyRunAsAny,
			},
		},
	}
}

// EmptyRoot returns an empty Root.
func (t *TestHelper) EmptyRoot() *ast.Root {
	return &ast.Root{
		ImportToken: t.ImportToken,
		LoadTime:    t.ImportTime,
	}
}

// ClusterPolicies returns a Root with only cluster policies.
func (t *TestHelper) ClusterPolicies() *ast.Root {
	return &ast.Root{
		Cluster:     t.AcmeCluster(),
		ImportToken: t.ImportToken,
		LoadTime:    t.ImportTime,
	}
}

// UnknownResource returns a custom resource without a corresponding CRD on the cluster.
func (t *TestHelper) UnknownResource() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "uknown.group",
		Version: "v1",
		Kind:    "Unknown",
	})
	return u
}

// AdminRoleBinding returns the role binding for the admin role.
func (t *TestHelper) AdminRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     "alice@acme.com",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "admin",
		},
	}
}

// PodReaderRole returns the contents of the pod-reader role.
func (t *TestHelper) PodReaderRole() *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-reader",
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"pods"},
		}},
	}
}

// PodReaderRoleBinding returns the role binding for the pod-reader role.
func (t *TestHelper) PodReaderRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     "bob@acme.com",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     "pod-reader",
		},
	}
}

// DeploymentReaderRole returns the contents of the deployment-reader role.
func (t *TestHelper) DeploymentReaderRole() *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "deployment-reader",
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"deployments"},
		}},
	}
}

// DeploymentReaderRoleBinding returns the rolebinding for deployment-reader role.
func (t *TestHelper) DeploymentReaderRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: rbacv1.GroupName,
				Kind:     rbacv1.UserKind,
				Name:     "bob@acme.com",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     "deployment-reader",
		},
	}
}

// AcmeResourceQuota returns the resource quota for Acme corp.
func (t *TestHelper) AcmeResourceQuota() *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "quota",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("5"),
			},
		},
	}
}

// FrontendResourceQuota returns the resource quota for the frontend namespace.
func (t *TestHelper) FrontendResourceQuota() *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "quota",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5"),
			},
		},
	}
}

// AcmeCluster returns the cluster info for Acme corp.
func (t *TestHelper) AcmeCluster() *ast.Cluster {
	return &ast.Cluster{
		Objects: ClusterObjectSets(
			t.NomosAdminClusterRole(),
			t.NomosAdminClusterRoleBinding(),
			t.NomosPodSecurityPolicy(),
		),
	}
}

// AcmeTree returns a tree of nodes for testing.
func (t *TestHelper) AcmeTree() *ast.TreeNode {
	return t.acmeTree()
}

func (t *TestHelper) acmeTree() *ast.TreeNode {
	return &ast.TreeNode{
		Type:     node.AbstractNamespace,
		Relative: nomospath.NewFakeRelative("namespaces"),
		Objects: ObjectSets(
			// TODO: remove RoleBinding once flattening transform is written.
			t.AdminRoleBinding(),
			t.AcmeResourceQuota(),
		),
		Children: []*ast.TreeNode{
			{
				Type:        node.Namespace,
				Relative:    nomospath.NewFakeRelative("namespaces/frontend"),
				Labels:      map[string]string{"environment": "prod"},
				Annotations: map[string]string{"has-waffles": "true"},
				Objects: ObjectSets(
					t.PodReaderRoleBinding(),
					t.PodReaderRole(),
					t.FrontendResourceQuota(),
				),
			},
			{
				Type:        node.Namespace,
				Relative:    nomospath.NewFakeRelative("namespaces/frontend-test"),
				Labels:      map[string]string{"environment": "test"},
				Annotations: map[string]string{"has-waffles": "false"},
				Objects: ObjectSets(
					t.DeploymentReaderRoleBinding(),
					t.DeploymentReaderRole(),
				),
			},
		},
	}
}

// ClusterRegistry returns the contents of a test cluster registry directory.
func (t *TestHelper) ClusterRegistry() *ast.ClusterRegistry {
	return &ast.ClusterRegistry{
		Objects: ClusterRegistryObjectSets(
			&clusterregistry.Cluster{},
		),
	}
}

// System returns the contents of a test system directory.
func (t *TestHelper) System() *ast.System {
	return &ast.System{
		Objects: SystemObjectSets(
			&v1alpha1.Repo{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
					Kind:       "Repo",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "nomos",
				},
				Spec: v1alpha1.RepoSpec{
					Version: "0.1.0",
				},
			},
		),
	}
}

// NamespacePolicies returns a Root with an example hierarchy.
func (t *TestHelper) NamespacePolicies() *ast.Root {
	return &ast.Root{
		Cluster:     &ast.Cluster{},
		Tree:        t.acmeTree(),
		ImportToken: t.ImportToken,
		LoadTime:    t.ImportTime,
	}
}

// AcmeRoot returns a Root with an example hierarchy.
func (t *TestHelper) AcmeRoot() *ast.Root {
	return &ast.Root{
		ClusterRegistry: t.ClusterRegistry(),
		System:          t.System(),
		Cluster:         t.AcmeCluster(),
		Tree:            t.acmeTree(),
		ImportToken:     t.ImportToken,
		LoadTime:        t.ImportTime,
	}
}
