package fake

import (
	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
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
func NamespaceSelectorObject(opts ...object.MetaMutator) *nomosv1.NamespaceSelector {
	result := &nomosv1.NamespaceSelector{
		TypeMeta: toTypeMeta(kinds.NamespaceSelector()),
	}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// NamespaceSelector returns a Nomos NamespaceSelector at a default path.
func NamespaceSelector(opts ...object.MetaMutator) ast.FileObject {
	return NamespaceSelectorAtPath("namespaces/ns.yaml", opts...)
}

// NamespaceSelectorAtPath returns a NamespaceSelector at the specified path.
func NamespaceSelectorAtPath(path string, opts ...object.MetaMutator) ast.FileObject {
	return fileObject(NamespaceSelectorObject(opts...), path)
}

// RoleBinding returns an RBAC RoleBinding.
func RoleBinding(opts ...object.MetaMutator) ast.FileObject {
	return RoleBindingAtPath("namespaces/foo/rolebinding.yaml", opts...)
}

// RoleBindingAtPath returns a RoleBinding at the specified path.
func RoleBindingAtPath(path string, opts ...object.MetaMutator) ast.FileObject {
	obj := &rbacv1alpha1.RoleBinding{TypeMeta: toTypeMeta(kinds.RoleBinding())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return fileObject(obj, path)
}

// ClusterRole returns an RBAC ClusterRole at the specified path.
func ClusterRole(opts ...object.MetaMutator) ast.FileObject {
	return ClusterRoleAtPath("cluster/cr.yaml", opts...)
}

// ClusterRoleAtPath returns a ClusterRole at the specified path.
func ClusterRoleAtPath(path string, opts ...object.MetaMutator) ast.FileObject {
	obj := &rbacv1alpha1.ClusterRole{TypeMeta: toTypeMeta(kinds.ClusterRole())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return fileObject(obj, path)
}

// ClusterSelector returns a Nomos ClusterSelector.
func ClusterSelector(opts ...object.MetaMutator) ast.FileObject {
	return ClusterSelectorAtPath("cluster/cs.yaml", opts...)
}

// ClusterSelectorAtPath returns a ClusterSelector at the specified path.
func ClusterSelectorAtPath(path string, opts ...object.MetaMutator) ast.FileObject {
	obj := &nomosv1.ClusterSelector{TypeMeta: toTypeMeta(kinds.ClusterSelector())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return fileObject(obj, path)
}

// Cluster returns a K8S Cluster resource in a fileObject.
func Cluster(opts ...object.MetaMutator) ast.FileObject {
	obj := &clusterregistry.Cluster{
		TypeMeta: toTypeMeta(kinds.Cluster()),
	}
	defaultMutate(obj)
	mutate(obj, opts...)

	return fileObject(obj, "clusterregistry/cluster.yaml")
}

// ClusterAtPath returns a Cluster at the specified path.
func ClusterAtPath(path string, opts ...object.MetaMutator) ast.FileObject {
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
func CustomResourceDefinitionObject(opts ...object.MetaMutator) *v1beta1.CustomResourceDefinition {
	result := &v1beta1.CustomResourceDefinition{
		TypeMeta: toTypeMeta(kinds.CustomResourceDefinition()),
	}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// CustomResourceDefinition returns a fileObject containing a CustomResourceDefinition at a
// default path.
func CustomResourceDefinition(opts ...object.MetaMutator) ast.FileObject {
	return fileObject(CustomResourceDefinitionObject(opts...), "cluster/crd.yaml")
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

	return fileObject(obj, path)
}

// SyncAtPath returns a nomos Sync at the specified path.
func SyncAtPath(path string) ast.FileObject {
	obj := &nomosv1.Sync{TypeMeta: toTypeMeta(kinds.Sync())}
	defaultMutate(obj)

	return fileObject(obj, path)
}

// PersistentVolumeObject returns a PersistentVolume Object.
func PersistentVolumeObject(opts ...object.MetaMutator) *corev1.PersistentVolume {
	result := &corev1.PersistentVolume{TypeMeta: toTypeMeta(kinds.PersistentVolume())}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// ReplicaSet returns a default ReplicaSet object.
func ReplicaSet(opts ...object.MetaMutator) ast.FileObject {
	obj := &appsv1.ReplicaSet{TypeMeta: toTypeMeta(kinds.ReplicaSet())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return fileObject(obj, "namespaces/foo/replicaset.yaml")
}

// RoleAtPath returns an initialized Role in a fileObject.
func RoleAtPath(path string, opts ...object.MetaMutator) ast.FileObject {
	obj := &rbac.Role{TypeMeta: toTypeMeta(kinds.Role())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return ast.NewFileObject(obj, cmpath.FromSlash(path))
}

// Role returns a Role with a default file path.
func Role(opts ...object.MetaMutator) ast.FileObject {
	return RoleAtPath("namespaces/foo/role.yaml", opts...)
}

func toTypeMeta(gvk schema.GroupVersionKind) v1.TypeMeta {
	return v1.TypeMeta{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
	}
}
