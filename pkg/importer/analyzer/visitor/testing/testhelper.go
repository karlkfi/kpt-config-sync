package testing

import (
	"time"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/util/repo"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

const (
	// ImportToken defines a default token to use for testing.
	ImportToken = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	// ClusterAdmin is the name of the test cluster admin role.
	ClusterAdmin = "cluster-admin"

	// ClusterAdminBinding is the name of the test cluster admin role binding.
	ClusterAdminBinding = "cluster-admin-binding"
)

var (
	// ImportTime defines a default time to use for testing.
	ImportTime = metav1.NewTime(time.Date(2017, 8, 10, 5, 16, 00, 0, time.FixedZone("PDT", -7*60*60)))
)

// ObjectSets constructs a list of ObjectSet from a list of runtime.Object.
func ObjectSets(runtimeObjs ...core.Object) []*ast.NamespaceObject {
	astObjs := make([]*ast.NamespaceObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.NamespaceObject{FileObject: *ast.ParseFileObject(runtimeObjs[idx])}
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
func ClusterObjectSets(runtimeObjs ...core.Object) []*ast.ClusterObject {
	astObjs := make([]*ast.ClusterObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.ClusterObject{FileObject: *ast.ParseFileObject(runtimeObjs[idx])}
	}
	return astObjs
}

// ClusterRegistryObjectSets constructs a list of ObjectSet from a list of runtime.NamespaceObject.
func ClusterRegistryObjectSets(runtimeObjs ...core.Object) []*ast.ClusterRegistryObject {
	astObjs := make([]*ast.ClusterRegistryObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.ClusterRegistryObject{FileObject: *ast.ParseFileObject(runtimeObjs[idx])}
	}
	return astObjs
}

// SystemObjectSets constructs a list of ObjectSet from a list of runtime.NamespaceObject.
func SystemObjectSets(runtimeObjs ...core.Object) []*ast.SystemObject {
	astObjs := make([]*ast.SystemObject, len(runtimeObjs))
	for idx := range runtimeObjs {
		astObjs[idx] = &ast.SystemObject{FileObject: *ast.ParseFileObject(runtimeObjs[idx])}
	}
	return astObjs
}

// Helper provides a number of pre-built types for use in testcases.  This does not set an Token
// or ImportTime
var Helper TestHelper

// TestHelper provides a number of pre-built types for use in testcases.
type TestHelper struct {
	ImportToken string
	ImportTime  time.Time
}

// NomosAdminClusterRole returns a ClusterRole for testing.
func (t *TestHelper) NomosAdminClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ClusterAdmin,
		},
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{rbacv1.VerbAll},
			APIGroups: []string{configmanagement.GroupName},
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
			Name: ClusterAdminBinding,
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
			Name:     ClusterAdmin,
		},
	}
}

// NomosPodSecurityPolicy returns a PodSecurityPolicy for testing.
func (t *TestHelper) NomosPodSecurityPolicy() *policyv1beta1.PodSecurityPolicy {
	return &policyv1beta1.PodSecurityPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyv1beta1.SchemeGroupVersion.String(),
			Kind:       "PodSecurityPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "example",
		},
		Spec: policyv1beta1.PodSecurityPolicySpec{
			Privileged: false,
			RunAsUser: policyv1beta1.RunAsUserStrategyOptions{
				Rule: policyv1beta1.RunAsUserStrategyRunAsAny,
			},
		},
	}
}

// CRD returns a CRD for testing.
func (t *TestHelper) CRD() *v1beta1.CustomResourceDefinition {
	return &v1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1beta1.SchemeGroupVersion.String(),
			Kind:       kinds.CustomResourceDefinitionV1Beta1().Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "example",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: "some.group",
			Versions: []v1beta1.CustomResourceDefinitionVersion{{
				Name:    "v1",
				Served:  true,
				Storage: true,
			}},
			Names: v1beta1.CustomResourceDefinitionNames{
				Plural: "names",
				Kind:   "Name",
			},
		},
	}
}

// EmptyRoot returns an empty Root.
func (t *TestHelper) EmptyRoot() *ast.Root {
	return &ast.Root{}
}

// ClusterConfigs returns a Root with only cluster configs.
func (t *TestHelper) ClusterConfigs() *ast.Root {
	return &ast.Root{
		ClusterObjects: t.AcmeCluster(),
	}
}

// CRDClusterConfig returns a Root with only the CRD Cluster Config.
func (t *TestHelper) CRDClusterConfig() *ast.Root {
	return &ast.Root{
		ClusterObjects: ClusterObjectSets(
			t.CRD(),
		),
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
			Name:      "pod-reader",
			Namespace: "frontend",
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
			Name:      "admin",
			Namespace: "frontend",
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
			Name:      "deployment-reader",
			Namespace: "frontend-test",
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
			Name:      "admin",
			Namespace: "frontend-test",
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
			Name:      "quota",
			Namespace: "frontend",
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("5"),
			},
		},
	}
}

// AcmeCluster returns the cluster info for Acme corp.
func (t *TestHelper) AcmeCluster() []*ast.ClusterObject {
	return ClusterObjectSets(
		t.NomosAdminClusterRole(),
		t.NomosAdminClusterRoleBinding(),
		t.NomosPodSecurityPolicy(),
	)
}

// AcmeTree returns a tree of nodes for testing.
func (t *TestHelper) AcmeTree() *ast.TreeNode {
	return t.acmeTree()
}

func (t *TestHelper) acmeTree() *ast.TreeNode {
	return &ast.TreeNode{
		Type: node.AbstractNamespace,
		Path: cmpath.FromSlash("namespaces"),
		Objects: ObjectSets(
			// TODO: remove RoleBinding once flattening transform is written.
			t.AdminRoleBinding(),
			t.AcmeResourceQuota(),
		),
		Children: []*ast.TreeNode{
			{
				Type: node.Namespace,
				Path: cmpath.FromSlash("namespaces/frontend"),
				Objects: ObjectSets(
					t.PodReaderRoleBinding(),
					t.PodReaderRole(),
					t.FrontendResourceQuota(),
				),
			},
			{
				Type: node.Namespace,
				Path: cmpath.FromSlash("namespaces/frontend-test"),
				Objects: ObjectSets(
					t.DeploymentReaderRoleBinding(),
					t.DeploymentReaderRole(),
				),
			},
		},
	}
}

// ClusterRegistry returns the contents of a test cluster registry directory.
func (t *TestHelper) ClusterRegistry() []*ast.ClusterRegistryObject {
	return ClusterRegistryObjectSets(
		&clusterregistry.Cluster{},
	)
}

// System returns the contents of a test system directory.
func (t *TestHelper) System() []*ast.SystemObject {
	return SystemObjectSets(
		&v1.Repo{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       configmanagement.RepoKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "repo",
			},
			Spec: v1.RepoSpec{
				Version: repo.CurrentVersion,
			},
		},
	)
}

// NamespaceConfigs returns a Root with an example hierarchy.
func (t *TestHelper) NamespaceConfigs() *ast.Root {
	return &ast.Root{
		Tree: t.acmeTree(),
	}
}

// AcmeRoot returns a Root with an example hierarchy.
func (t *TestHelper) AcmeRoot() *ast.Root {
	return &ast.Root{
		ClusterRegistryObjects: t.ClusterRegistry(),
		SystemObjects:          t.System(),
		ClusterObjects:         t.AcmeCluster(),
		Tree:                   t.acmeTree(),
	}
}
