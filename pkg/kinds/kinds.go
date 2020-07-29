package kinds

import (
	"fmt"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Anvil returns the GroupVersionKind for Anvil Custom Resource used in tests.
func Anvil() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "acme.com",
		Version: "v1",
		Kind:    "Anvil",
	}
}

// Sync returns the canonical Sync GroupVersionKind
func Sync() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind(configmanagement.SyncKind)
}

// RoleBinding returns the canonical RoleBinding GroupVersionKind
func RoleBinding() schema.GroupVersionKind {
	return rbacv1.SchemeGroupVersion.WithKind("RoleBinding")
}

// Role returns the canonical Role GroupVersionKind
func Role() schema.GroupVersionKind {
	return rbacv1.SchemeGroupVersion.WithKind("Role")
}

// ResourceQuota returns the canonical ResourceQuota GroupVersionKind
func ResourceQuota() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("ResourceQuota")
}

// Repo returns the canonical Repo GroupVersionKind
func Repo() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind(configmanagement.RepoKind)
}

// PersistentVolume returns the canonical PersistentVolume GroupVersionKind
func PersistentVolume() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("PersistentVolume")
}

// NamespaceConfig returns the canonical NamespaceConfig GroupVersionKind
func NamespaceConfig() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind(configmanagement.NamespaceConfigKind)
}

// PodSecurityPolicy returns the canonical PodSecurityPolicy GroupVersionKind
func PodSecurityPolicy() schema.GroupVersionKind {
	return policyv1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy")
}

// NamespaceSelector returns the canonical NamespaceSelector GroupVersionKind
func NamespaceSelector() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind(configmanagement.NamespaceSelectorKind)
}

// Namespace returns the canonical Namespace GroupVersionKind
func Namespace() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Namespace")
}

// CustomResourceDefinitionKind is the Kind for CustomResourceDefinitions
const CustomResourceDefinitionKind = "CustomResourceDefinition"

// CustomResourceDefinitionV1Beta1 returns the v1beta1 CustomResourceDefinition GroupVersionKind
func CustomResourceDefinitionV1Beta1() schema.GroupVersionKind {
	return CustomResourceDefinition().WithVersion(v1beta1.SchemeGroupVersion.Version)
}

// CustomResourceDefinitionV1 returns the v1 CustomResourceDefinition GroupVersionKind
func CustomResourceDefinitionV1() schema.GroupVersionKind {
	return CustomResourceDefinition().WithVersion("v1")
}

// CustomResourceDefinition returns the canonical CustomResourceDefinition GroupKind
func CustomResourceDefinition() schema.GroupKind {
	return schema.GroupKind{
		Group: v1beta1.GroupName,
		Kind:  CustomResourceDefinitionKind,
	}
}

// ClusterSelector returns the canonical ClusterSelector GroupVersionKind
func ClusterSelector() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind(configmanagement.ClusterSelectorKind)
}

// ClusterRoleBinding returns the canonical ClusterRoleBinding GroupVersionKind
func ClusterRoleBinding() schema.GroupVersionKind {
	return rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding")
}

// ClusterRoleBindingV1Beta1 returns the canonical ClusterRoleBinding GroupVersionKind
func ClusterRoleBindingV1Beta1() schema.GroupVersionKind {
	return rbacv1beta1.SchemeGroupVersion.WithKind("ClusterRoleBinding")
}

// ClusterRole returns the canonical ClusterRole GroupVersionKind
func ClusterRole() schema.GroupVersionKind {
	return rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
}

// ClusterConfig returns the canonical ClusterConfig GroupVersionKind
func ClusterConfig() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind(configmanagement.ClusterConfigKind)
}

// Cluster returns the canonical Cluster GroupVersionKind
func Cluster() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "clusterregistry.k8s.io", Version: "v1alpha1", Kind: "Cluster"}
}

// Deployment returns the canonical Deployment GroupVersionKind
func Deployment() schema.GroupVersionKind {
	return appsv1.SchemeGroupVersion.WithKind("Deployment")
}

// DaemonSet returns the canonical DaemonSet GroupVersionKind
func DaemonSet() schema.GroupVersionKind {
	return appsv1.SchemeGroupVersion.WithKind("DaemonSet")
}

// ReplicaSet returns the canonical ReplicaSet GroupVersionKind
func ReplicaSet() schema.GroupVersionKind {
	return appsv1.SchemeGroupVersion.WithKind("ReplicaSet")
}

// HierarchyConfig returns the canonical HierarchyConfig GroupVersionKind
func HierarchyConfig() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind(configmanagement.HierarchyConfigKind)
}

// NetworkPolicy returns the canonical NetworkPolicy GroupVersionKind
func NetworkPolicy() schema.GroupVersionKind {
	return networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy")
}

// ConfigMap returns the canconical ConfigMap GroupVersionKind
func ConfigMap() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("ConfigMap")
}

// StatefulSet returns the canonical StatefulSet GroupVersionKind
func StatefulSet() schema.GroupVersionKind {
	return appsv1.SchemeGroupVersion.WithKind("StatefulSet")
}

// ConfigManagement returns the GroupVersionKind for ConfigManagement, an object
// that does not have other representation than a CRD in the operator library.
func ConfigManagement() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "configmanagement.gke.io",
		Version: "v1",
		Kind:    "ConfigManagement",
	}
}

// RepoSync returns the canonical RepoSync GroupVersionKind.
func RepoSync() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("RepoSync")
}

// RootSync returns the canonical RootSync GroupVersionKind.
func RootSync() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("RootSync")
}

// ResourceString returns a string describing the GroupVersionKind using fields specified in Kubernetes Objects.
func ResourceString(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf("apiVersion=%s/%s, kind=%s", gvk.Group, gvk.Version, gvk.Kind)
}

// Service returns the canonical Service GroupVersionKind.
func Service() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Service")
}
