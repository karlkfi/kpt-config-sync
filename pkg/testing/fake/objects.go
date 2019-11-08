package fake

import (
	"strings"

	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/kubernetes/pkg/apis/rbac"
)

// NamespaceSelectorObject returns an initialized NamespaceSelector.
func NamespaceSelectorObject(opts ...core.MetaMutator) *nomosv1.NamespaceSelector {
	result := &nomosv1.NamespaceSelector{
		TypeMeta: toTypeMeta(kinds.NamespaceSelector()),
	}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// NamespaceSelector returns a Nomos NamespaceSelector at a default path.
func NamespaceSelector(opts ...core.MetaMutator) ast.FileObject {
	return NamespaceSelectorAtPath("namespaces/ns.yaml", opts...)
}

// NamespaceSelectorAtPath returns a NamespaceSelector at the specified path.
func NamespaceSelectorAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(NamespaceSelectorObject(opts...), path)
}

// ResourceQuotaObject initializes a ResouceQuota.
func ResourceQuotaObject(opts ...core.MetaMutator) *corev1.ResourceQuota {
	obj := &corev1.ResourceQuota{TypeMeta: toTypeMeta(kinds.ResourceQuota())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// RoleBindingObject initializes a RoleBinding.
func RoleBindingObject(opts ...core.MetaMutator) *rbacv1alpha1.RoleBinding {
	obj := &rbacv1alpha1.RoleBinding{TypeMeta: toTypeMeta(kinds.RoleBinding())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// RoleBinding returns an RBAC RoleBinding.
func RoleBinding(opts ...core.MetaMutator) ast.FileObject {
	return RoleBindingAtPath("namespaces/foo/rolebinding.yaml", opts...)
}

// RoleBindingAtPath returns a RoleBinding at the specified path.
func RoleBindingAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(RoleBindingObject(opts...), path)
}

// ClusterRole returns an RBAC ClusterRole at the specified path.
func ClusterRole(opts ...core.MetaMutator) ast.FileObject {
	return ClusterRoleAtPath("cluster/cr.yaml", opts...)
}

// ClusterRoleBindingAtPath returns a ClusterRoleBinding at the specified path.
func ClusterRoleBindingAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	obj := &rbacv1alpha1.ClusterRoleBinding{TypeMeta: toTypeMeta(kinds.ClusterRoleBinding())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return FileObject(obj, path)
}

// ClusterRoleBindingObject initializes a ClusterRoleBinding.
func ClusterRoleBindingObject(opts ...core.MetaMutator) *rbacv1alpha1.ClusterRoleBinding {
	obj := &rbacv1alpha1.ClusterRoleBinding{TypeMeta: toTypeMeta(kinds.ClusterRoleBinding())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// ClusterRoleBinding returns an initialized ClusterRoleBinding.
func ClusterRoleBinding(opts ...core.MetaMutator) ast.FileObject {
	return ClusterRoleBindingAtPath("cluster/crb.yaml", opts...)
}

// ClusterRoleAtPath returns a ClusterRole at the specified path.
func ClusterRoleAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	obj := &rbacv1alpha1.ClusterRole{TypeMeta: toTypeMeta(kinds.ClusterRole())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return FileObject(obj, path)
}

// ClusterSelectorObject initializes a ClusterSelector object.
func ClusterSelectorObject(opts ...core.MetaMutator) *nomosv1.ClusterSelector {
	obj := &nomosv1.ClusterSelector{TypeMeta: toTypeMeta(kinds.ClusterSelector())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// ClusterSelector returns a Nomos ClusterSelector.
func ClusterSelector(opts ...core.MetaMutator) ast.FileObject {
	return ClusterSelectorAtPath("cluster/cs.yaml", opts...)
}

// ClusterSelectorAtPath returns a ClusterSelector at the specified path.
func ClusterSelectorAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(ClusterSelectorObject(opts...), path)
}

// Cluster returns a K8S Cluster resource in a FileObject.
func Cluster(opts ...core.MetaMutator) ast.FileObject {
	obj := &clusterregistry.Cluster{
		TypeMeta: toTypeMeta(kinds.Cluster()),
	}
	defaultMutate(obj)
	mutate(obj, opts...)

	return FileObject(obj, "clusterregistry/cluster.yaml")
}

// ClusterAtPath returns a Cluster at the specified path.
func ClusterAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	result := Cluster(opts...)
	result.Path = cmpath.FromSlash(path)
	return result
}

// ConfigManagement returns a fake ConfigManagement.
func ConfigManagement(path string) ast.FileObject {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(kinds.ConfigManagement())
	return ast.NewFileObject(u, cmpath.FromSlash(path))
}

// CustomResourceDefinitionObject returns an initialized CustomResourceDefinition.
func CustomResourceDefinitionObject(opts ...core.MetaMutator) *v1beta1.CustomResourceDefinition {
	result := &v1beta1.CustomResourceDefinition{
		TypeMeta: toTypeMeta(kinds.CustomResourceDefinition()),
	}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// CustomResourceDefinition returns a FileObject containing a CustomResourceDefinition at a
// default path.
func CustomResourceDefinition(opts ...core.MetaMutator) ast.FileObject {
	return FileObject(CustomResourceDefinitionObject(opts...), "cluster/crd.yaml")
}

// AnvilAtPath returns an Anvil Custom Resource.
func AnvilAtPath(path string) ast.FileObject {
	obj := &v1beta1.CustomResourceDefinition{
		TypeMeta: toTypeMeta(kinds.Anvil()),
		ObjectMeta: v1.ObjectMeta{
			Name: "anvil",
		},
	}
	defaultMutate(obj)

	return FileObject(obj, path)
}

// SyncObject returns a Sync configured for a particular
func SyncObject(gk schema.GroupKind, opts ...core.MetaMutator) *nomosv1.Sync {
	obj := &nomosv1.Sync{TypeMeta: toTypeMeta(kinds.Sync())}
	obj.Name = strings.ToLower(gk.String())
	obj.ObjectMeta.Finalizers = append(obj.ObjectMeta.Finalizers, nomosv1.SyncFinalizer)
	obj.Spec.Group = gk.Group
	obj.Spec.Kind = gk.Kind

	mutate(obj, opts...)
	return obj
}

// SyncAtPath returns a nomos Sync at the specified path.
func SyncAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(SyncObject(kinds.Role().GroupKind(), opts...), path)
}

// PersistentVolumeObject returns a PersistentVolume Object.
func PersistentVolumeObject(opts ...core.MetaMutator) *corev1.PersistentVolume {
	result := &corev1.PersistentVolume{TypeMeta: toTypeMeta(kinds.PersistentVolume())}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// ReplicaSet returns a default ReplicaSet object.
func ReplicaSet(opts ...core.MetaMutator) ast.FileObject {
	obj := &appsv1.ReplicaSet{TypeMeta: toTypeMeta(kinds.ReplicaSet())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return FileObject(obj, "namespaces/foo/replicaset.yaml")
}

// RoleObject initializes a Role.
func RoleObject(opts ...core.MetaMutator) *rbac.Role {
	obj := &rbac.Role{TypeMeta: toTypeMeta(kinds.Role())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// RoleAtPath returns an initialized Role in a FileObject.
func RoleAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return ast.NewFileObject(RoleObject(opts...), cmpath.FromSlash(path))
}

// Role returns a Role with a default file path.
func Role(opts ...core.MetaMutator) ast.FileObject {
	return RoleAtPath("namespaces/foo/role.yaml", opts...)
}

// ConfigMapObject returns an initialized ConfigMap.
func ConfigMapObject(opts ...core.MetaMutator) *corev1.ConfigMap {
	obj := &corev1.ConfigMap{TypeMeta: toTypeMeta(kinds.ConfigMap())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// ConfigMapAtPath returns a ConfigMap at the specified filepath.
func ConfigMapAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(ConfigMapObject(opts...), path)
}

func toTypeMeta(gvk schema.GroupVersionKind) v1.TypeMeta {
	return v1.TypeMeta{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
	}
}
