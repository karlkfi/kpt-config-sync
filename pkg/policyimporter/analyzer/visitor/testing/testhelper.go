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
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ObjectSets constructs a list of ObjectSet from a list of runtime.Object.
func ObjectSets(runtimeObjs ...runtime.Object) ast.ObjectList {
	astObjs := make([]*ast.NamespaceObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.NamespaceObject{FileObject: ast.FileObject{Object: runtimeObjs[idx]}}
	}
	return ast.ObjectList(astObjs)
}

// FileObjectSets constructs a list of ObjectSet from a list of ast.FileObject
func FileObjectSets(runtimeObjs ...ast.FileObject) ast.ObjectList {
	astObjs := make([]*ast.NamespaceObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.NamespaceObject{FileObject: runtimeObjs[idx]}
	}
	return ast.ObjectList(astObjs)
}

// ClusterObjectSets constructs a list of ObjectSet from a list of runtime.NamespaceObject.
func ClusterObjectSets(runtimeObjs ...runtime.Object) ast.ClusterObjectList {
	astObjs := make([]*ast.ClusterObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.ClusterObject{FileObject: ast.FileObject{Object: runtimeObjs[idx]}}
	}
	return ast.ClusterObjectList(astObjs)
}

// Helper provides a number of pre-built types for use in testcases.
var Helper TestHelper

// TestHelper provides a number of pre-built types for use in testcases.
type TestHelper struct {
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
		Rules: []rbacv1.PolicyRule{rbacv1.PolicyRule{
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
			rbacv1.Subject{
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

// EmptyContext returns an empty git context.
func (t *TestHelper) EmptyContext() *ast.Context {
	return &ast.Context{}
}

// ClusterPolicies returns a GitContext with only cluster policies.
func (t *TestHelper) ClusterPolicies() *ast.Context {
	return &ast.Context{Cluster: t.AcmeCluster()}
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
			rbacv1.Subject{
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
		Rules: []rbacv1.PolicyRule{rbacv1.PolicyRule{
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
			rbacv1.Subject{
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
		Rules: []rbacv1.PolicyRule{rbacv1.PolicyRule{
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
			rbacv1.Subject{
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

// ReservedNamespaces returns a GitContext with only reserved namespaces.
func (t *TestHelper) ReservedNamespaces() *ast.Context {
	return &ast.Context{
		Cluster:            &ast.Cluster{},
		ReservedNamespaces: t.AcmeReserved(),
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

// AcmeReserved returns the reserved namespaces for Acme corp.
func (t *TestHelper) AcmeReserved() *ast.ReservedNamespaces {
	return &ast.ReservedNamespaces{
		ConfigMap: corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: v1alpha1.ReservedNamespacesConfigMapName,
			},
			Data: map[string]string{
				"testing":      string(v1alpha1.ReservedAttribute),
				"more-testing": string(v1alpha1.ReservedAttribute),
			},
		},
	}
}

func (t *TestHelper) acmeTree() *ast.TreeNode {
	return &ast.TreeNode{
		Type: ast.Policyspace,
		Path: "acme",
		Objects: ObjectSets(
			// TODO: remove RoleBinding once flattening transform is written.
			t.AdminRoleBinding(),
			t.AcmeResourceQuota(),
		),
		Children: []*ast.TreeNode{
			&ast.TreeNode{
				Type:        ast.Namespace,
				Path:        "acme/frontend",
				Labels:      map[string]string{"environment": "prod"},
				Annotations: map[string]string{"has-waffles": "true"},
				Objects: ObjectSets(
					t.PodReaderRoleBinding(),
					t.PodReaderRole(),
					t.FrontendResourceQuota(),
				),
			},
			&ast.TreeNode{
				Type:        ast.Namespace,
				Path:        "acme/frontend-test",
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

// NamespacePolicies returns a GitContext with an example hierarchy.
func (t *TestHelper) NamespacePolicies() *ast.Context {
	return &ast.Context{
		Cluster: &ast.Cluster{},
		Tree:    t.acmeTree(),
	}
}

// AcmeContext returns a GitContext with an example hierarchy.
func (t *TestHelper) AcmeContext() *ast.Context {
	return &ast.Context{
		Cluster:            t.AcmeCluster(),
		ReservedNamespaces: t.AcmeReserved(),
		Tree:               t.acmeTree(),
	}
}
