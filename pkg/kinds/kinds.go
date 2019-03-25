package kinds

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	oidcconfig "github.com/google/nomos/pkg/oidc/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/scale/scheme/extensionsv1beta1"
)

// Sync returns the canonical Sync GroupVersionKind
func Sync() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("Sync")
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
	return v1.SchemeGroupVersion.WithKind("Repo")
}

// PersistentVolume returns the canonical PersistentVolume GroupVersionKind
func PersistentVolume() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("PersistentVolume")
}

// NamespaceConfig returns the canonical NamespaceConfig GroupVersionKind
func NamespaceConfig() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("NamespaceConfig")
}

// PodSecurityPolicy returns the canonical PodSecurityPolicy GroupVersionKind
func PodSecurityPolicy() schema.GroupVersionKind {
	return policyv1beta1.SchemeGroupVersion.WithKind("PodSecurityPolicy")
}

// NamespaceSelector returns the canonical NamespaceSelector GroupVersionKind
func NamespaceSelector() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("NamespaceSelector")
}

// Namespace returns the canonical Namespace GroupVersionKind
func Namespace() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("Namespace")
}

// CustomResourceDefinition returns the canonical CustomResourceDefinition GroupVersionKind
func CustomResourceDefinition() schema.GroupVersionKind {
	return extensionsv1beta1.SchemeGroupVersion.WithKind("CustomResourceDefinition")
}

// ClusterSelector returns the canonical ClusterSelector GroupVersionKind
func ClusterSelector() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("ClusterSelector")
}

// ClusterRoleBinding returns the canonical ClusterRoleBinding GroupVersionKind
func ClusterRoleBinding() schema.GroupVersionKind {
	return rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding")
}

// ClusterRole returns the canonical ClusterRole GroupVersionKind
func ClusterRole() schema.GroupVersionKind {
	return rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
}

// ClusterConfig returns the canonical ClusterConfig GroupVersionKind
func ClusterConfig() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("ClusterConfig")
}

// Cluster returns the canonical Cluster GroupVersionKind
func Cluster() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "clusterregistry.k8s.io", Version: "v1alpha1", Kind: "Cluster"}
}

// ClientID returns the canonical ClientID GroupVersionKind
func ClientID() schema.GroupVersionKind {
	return oidcconfig.SchemeGroupVersion.WithKind("ClientID")
}

// Deployment returns the canonical Deployment GroupVersionKind
func Deployment() schema.GroupVersionKind {
	return appsv1.SchemeGroupVersion.WithKind("Deployment")
}

// HierarchyConfig returns the canonical HierarchyConfig GroupVersionKind
func HierarchyConfig() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("HierarchyConfig")
}

// HierarchicalQuota returns the canonical HierarchyConfig GroupVersionKind
func HierarchicalQuota() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("HierarchicalQuota")
}

// ResourceString returns a string describing the GroupVersionKind using fields specified in Kubernetes Objects.
func ResourceString(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf("apiVersion=%s/%s, kind=%s", gvk.Group, gvk.Version, gvk.Kind)
}
