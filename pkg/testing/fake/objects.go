package fake

import (
	"encoding/json"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// NamespaceSelectorObject returns an initialized NamespaceSelector.
func NamespaceSelectorObject(opts ...core.MetaMutator) *v1.NamespaceSelector {
	result := &v1.NamespaceSelector{
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

// NamespaceSelectorAtPathWithName returns a NamespaceSelector at the specified
// path with the specified name.
func NamespaceSelectorAtPathWithName(path string, name string, opts ...core.MetaMutator) ast.FileObject {
	opts = append(opts, core.Name(name))
	return FileObject(NamespaceSelectorObject(opts...), path)
}

// ResourceQuotaObject initializes a ResouceQuota.
func ResourceQuotaObject(opts ...core.MetaMutator) *corev1.ResourceQuota {
	obj := &corev1.ResourceQuota{TypeMeta: toTypeMeta(kinds.ResourceQuota())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// ResourceQuotaObjectUnstructured initializes a ResouceQuota as an unstructured object.
func ResourceQuotaObjectUnstructured(opts ...core.MetaMutator) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(kinds.ResourceQuota())
	mutate(u, opts...)
	return u
}

// ResourceQuota initializes a FileObject with a ResourceQuota.
func ResourceQuota(opts ...core.MetaMutator) ast.FileObject {
	return FileObject(ResourceQuotaObject(opts...), "namespaces/foo/rq.yaml")
}

// RoleBindingObject initializes a RoleBinding.
func RoleBindingObject(opts ...core.MetaMutator) *rbacv1.RoleBinding {
	obj := &rbacv1.RoleBinding{TypeMeta: toTypeMeta(kinds.RoleBinding())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// RoleBinding returns an rbacv1 RoleBinding.
func RoleBinding(opts ...core.MetaMutator) ast.FileObject {
	return RoleBindingAtPath("namespaces/foo/rolebinding.yaml", opts...)
}

// RoleBindingAtPath returns a RoleBinding at the specified path.
func RoleBindingAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(RoleBindingObject(opts...), path)
}

// ClusterRole returns an rbacv1 ClusterRole at the specified path.
func ClusterRole(opts ...core.MetaMutator) ast.FileObject {
	return ClusterRoleAtPath("cluster/cr.yaml", opts...)
}

// ClusterRoleBindingAtPath returns a ClusterRoleBinding at the specified path.
func ClusterRoleBindingAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	obj := &rbacv1.ClusterRoleBinding{TypeMeta: toTypeMeta(kinds.ClusterRoleBinding())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return FileObject(obj, path)
}

// ClusterRoleBindingObject initializes a ClusterRoleBinding.
func ClusterRoleBindingObject(opts ...core.MetaMutator) *rbacv1.ClusterRoleBinding {
	obj := &rbacv1.ClusterRoleBinding{TypeMeta: toTypeMeta(kinds.ClusterRoleBinding())}
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
	obj := &rbacv1.ClusterRole{TypeMeta: toTypeMeta(kinds.ClusterRole())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return FileObject(obj, path)
}

// ClusterSelectorObject initializes a ClusterSelector object.
func ClusterSelectorObject(opts ...core.MetaMutator) *v1.ClusterSelector {
	obj := &v1.ClusterSelector{TypeMeta: toTypeMeta(kinds.ClusterSelector())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// ClusterSelector returns a Nomos ClusterSelector.
func ClusterSelector(opts ...core.MetaMutator) ast.FileObject {
	return clusterSelectorAtPath("cluster/cs.yaml", opts...)
}

// ClusterSelectorAtPath returns a ClusterSelector at the specified path.
func clusterSelectorAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
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

// CustomResourceDefinitionV1Beta1Object returns an initialized CustomResourceDefinition.
func CustomResourceDefinitionV1Beta1Object(opts ...core.MetaMutator) *v1beta1.CustomResourceDefinition {
	result := &v1beta1.CustomResourceDefinition{
		TypeMeta: toTypeMeta(kinds.CustomResourceDefinitionV1Beta1()),
	}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// CustomResourceDefinitionV1Beta1 returns a FileObject containing a
// CustomResourceDefinition at a default path.
func CustomResourceDefinitionV1Beta1(opts ...core.MetaMutator) ast.FileObject {
	return FileObject(CustomResourceDefinitionV1Beta1Object(opts...), "cluster/crd.yaml")
}

// CustomResourceDefinitionV1Beta1Unstructured returns a v1Beta1 CRD as an unstructured
func CustomResourceDefinitionV1Beta1Unstructured(opts ...core.MetaMutator) *unstructured.Unstructured {
	o := CustomResourceDefinitionV1Beta1Object(opts...)
	jsn, err := json.Marshal(o)
	if err != nil {
		// Should be impossible, and this is test-only code so it's fine.
		panic(err)
	}
	u := &unstructured.Unstructured{}
	err = json.Unmarshal(jsn, u)
	u.SetGroupVersionKind(kinds.CustomResourceDefinitionV1Beta1())
	if err != nil {
		// Should be impossible, and this is test-only code so it's fine.
		panic(err)
	}
	return u
}

// ToCustomResourceDefinitionV1Object converts a v1beta1.CustomResourceDefinition
// to an Unstructured masquerading as a v1.CRD.
func ToCustomResourceDefinitionV1Object(o *v1beta1.CustomResourceDefinition) *unstructured.Unstructured {
	jsn, err := json.Marshal(o)
	if err != nil {
		// Should be impossible, and this is test-only code so it's fine.
		panic(err)
	}
	u := &unstructured.Unstructured{}
	err = json.Unmarshal(jsn, u)
	u.SetGroupVersionKind(kinds.CustomResourceDefinitionV1())
	if err != nil {
		// Should be impossible, and this is test-only code so it's fine.
		panic(err)
	}
	return u
}

// ToCustomResourceDefinitionV1 converts the type inside a FileObject into an
// unstructured.Unstructured masquerading as a
func ToCustomResourceDefinitionV1(o ast.FileObject) ast.FileObject {
	// This will panic if o.Object isn't a v1beta1.CRD, but this is what we want
	// and this is test code so it's fine.
	crd := o.Object.(*v1beta1.CustomResourceDefinition)
	o.Object = ToCustomResourceDefinitionV1Object(crd)
	return o
}

// AnvilAtPath returns an Anvil Custom Resource.
func AnvilAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	obj := &v1beta1.CustomResourceDefinition{
		TypeMeta: toTypeMeta(kinds.Anvil()),
		ObjectMeta: metav1.ObjectMeta{
			Name: "anvil",
		},
	}
	defaultMutate(obj)
	mutate(obj, opts...)

	return FileObject(obj, path)
}

// SyncObject returns a Sync configured for a particular
func SyncObject(gk schema.GroupKind, opts ...core.MetaMutator) *v1.Sync {
	obj := &v1.Sync{TypeMeta: toTypeMeta(kinds.Sync())}
	obj.Name = strings.ToLower(gk.String())
	obj.ObjectMeta.Finalizers = append(obj.ObjectMeta.Finalizers, v1.SyncFinalizer)
	obj.Spec.Group = gk.Group
	obj.Spec.Kind = gk.Kind

	mutate(obj, opts...)
	return obj
}

// PersistentVolumeObject returns a PersistentVolume Object.
func PersistentVolumeObject(opts ...core.MetaMutator) *corev1.PersistentVolume {
	result := &corev1.PersistentVolume{TypeMeta: toTypeMeta(kinds.PersistentVolume())}
	defaultMutate(result)
	mutate(result, opts...)

	return result
}

// RoleObject initializes a Role.
func RoleObject(opts ...core.MetaMutator) *rbacv1.Role {
	obj := &rbacv1.Role{TypeMeta: toTypeMeta(kinds.Role())}
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

// RoleUnstructuredAtPath returns an unstructured initialized Role in a FileObject.
func RoleUnstructuredAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(kinds.Role())
	mutate(u, opts...)
	return ast.NewFileObject(u, cmpath.FromSlash(path))
}

// ConfigMapObject returns an initialized ConfigMap.
func configMapObject(opts ...core.MetaMutator) *corev1.ConfigMap {
	obj := &corev1.ConfigMap{TypeMeta: toTypeMeta(kinds.ConfigMap())}
	defaultMutate(obj)
	mutate(obj, opts...)

	return obj
}

// ConfigMapAtPath returns a ConfigMap at the specified filepath.
func ConfigMapAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(configMapObject(opts...), path)
}

func toTypeMeta(gvk schema.GroupVersionKind) metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
	}
}
