package kinds

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	oidcconfig "github.com/google/nomos/pkg/oidc/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
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
	return v1beta1.SchemeGroupVersion.WithKind("CustomResourceDefinition")
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
	return v1.SchemeGroupVersion.WithKind("HierarchyConfig")
}

// HierarchicalQuota returns the canonical HierarchyConfig GroupVersionKind
func HierarchicalQuota() schema.GroupVersionKind {
	return v1.SchemeGroupVersion.WithKind("HierarchicalQuota")
}

// NetworkPolicy returns the canonical NetworkPolicy GroupVersionKind
func NetworkPolicy() schema.GroupVersionKind {
	return networkingv1.SchemeGroupVersion.WithKind("NetworkPolicy")
}

// ConfigMap returns the canconical ConfigMap GroupVersionKind
func ConfigMap() schema.GroupVersionKind {
	return corev1.SchemeGroupVersion.WithKind("ConfigMap")
}

// ConfigManagement returns the GroupVersionKind for ConfigManagement, an object
// that does not have other representation than a CRD in the operator library.
func ConfigManagement() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "addons.sigs.k8s.io",
		Version: "v1alpha1",
		Kind:    "ConfigManagement",
	}
}

// ResourceString returns a string describing the GroupVersionKind using fields specified in Kubernetes Objects.
func ResourceString(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf("apiVersion=%s/%s, kind=%s", gvk.Group, gvk.Version, gvk.Kind)
}

// Organization returns the Group and Kind of Organizations.
func Organization() schema.GroupKind {
	return schema.GroupKind{
		// TODO(b/134173753) Align with Nomos team about value of Group, same below.
		Group: "bespin.dev",
		Kind:  "Organization",
	}
}

// Folder returns the Group and Kind of Folders.
func Folder() schema.GroupKind {
	return schema.GroupKind{
		Group: "bespin.dev",
		Kind:  "Folder",
	}
}

// Project returns the Group and Kind of Projects.
func Project() schema.GroupKind {
	return schema.GroupKind{
		Group: "bespin.dev",
		Kind:  "Project",
	}
}

// IAMPolicy returns the Group and Kind of IAMPolicies.
func IAMPolicy() schema.GroupKind {
	return schema.GroupKind{
		Group: "bespin.dev",
		Kind:  "IAMPolicy",
	}
}

// OrganizationPolicy returns the Group and Kind of OrganizationPolicies.
func OrganizationPolicy() schema.GroupKind {
	return schema.GroupKind{
		Group: "bespin.dev",
		Kind:  "OrganizationPolicy",
	}
}
