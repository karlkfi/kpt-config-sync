package fake

import (
	v1alpha1 "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	corev1 "k8s.io/api/core/v1"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// Namespace returns a Kubernetes Namespace resource at the specified path.
// Initializes with metadata.name set to the correct name.
func Namespace(path string) ast.FileObject {
	relative := nomospath.NewFakeRelative(path)
	return ast.FileObject{
		Relative: relative,
		Object: &corev1.Namespace{
			TypeMeta: toTypeMeta(kinds.Namespace()),
			ObjectMeta: v1.ObjectMeta{
				Name: relative.Dir().Base(),
			},
		}}
}

// NamespaceSelector returns a Nomos NamespaceSelector at the specified path.
func NamespaceSelector(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &v1alpha1.NamespaceSelector{
			TypeMeta: toTypeMeta(kinds.NamespaceSelector()),
		},
	}
}

// Role returns an RBAC Role at the specified path.
func Role(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &rbacv1alpha1.Role{
			TypeMeta: toTypeMeta(kinds.Role()),
		},
	}
}

// RoleBinding returns an RBAC RoleBinding at the specified path.
func RoleBinding(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &rbacv1alpha1.RoleBinding{
			TypeMeta: toTypeMeta(kinds.RoleBinding()),
		},
	}
}

// ClusterRole returns an RBAC ClusterRole at the specified path.
func ClusterRole(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &rbacv1alpha1.ClusterRole{
			TypeMeta: toTypeMeta(kinds.ClusterRole()),
		},
	}
}

// ClusterSelector returns a Nomos ClusterSelector at the specified path.
func ClusterSelector(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &v1alpha1.ClusterSelector{
			TypeMeta: toTypeMeta(kinds.ClusterSelector()),
		},
	}
}

// Cluster returns a K8S Cluster resource at the specified path.
func Cluster(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &clusterregistry.Cluster{
			TypeMeta: toTypeMeta(kinds.Cluster()),
		},
	}
}

// Repo returns a nomos Repo at the specified path.
func Repo(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &v1alpha1.Repo{
			TypeMeta: toTypeMeta(kinds.Repo()),
		},
	}
}

// HierarchyConfig returns an empty HierarchyConfig at the specified path.
func HierarchyConfig(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &v1alpha1.HierarchyConfig{
			TypeMeta: toTypeMeta(kinds.HierarchyConfig()),
		},
	}
}

// Sync returns a nomos Sync at the specified path.
func Sync(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &v1alpha1.Sync{
			TypeMeta: toTypeMeta(kinds.Sync()),
		},
	}
}

func toTypeMeta(gvk schema.GroupVersionKind) v1.TypeMeta {
	return v1.TypeMeta{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
	}
}
