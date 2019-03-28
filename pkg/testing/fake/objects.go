package fake

import (
	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// Namespace returns a Kubernetes Namespace resource at the specified path.
// Initializes with metadata.name set to the correct name.
func Namespace(path string) ast.FileObject {
	relative := cmpath.FromSlash(path)
	return ast.FileObject{
		Path: relative,
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
		Path: cmpath.FromSlash(path),
		Object: &nomosv1.NamespaceSelector{
			TypeMeta: toTypeMeta(kinds.NamespaceSelector()),
		},
	}
}

// Role returns an RBAC Role at the specified path.
func Role(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &rbacv1alpha1.Role{
			TypeMeta: toTypeMeta(kinds.Role()),
		},
	}
}

// RoleBinding returns an RBAC RoleBinding at the specified path.
func RoleBinding(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &rbacv1alpha1.RoleBinding{
			TypeMeta: toTypeMeta(kinds.RoleBinding()),
		},
	}
}

// ClusterRole returns an RBAC ClusterRole at the specified path.
func ClusterRole(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &rbacv1alpha1.ClusterRole{
			TypeMeta: toTypeMeta(kinds.ClusterRole()),
		},
	}
}

// ClusterSelector returns a Nomos ClusterSelector at the specified path.
func ClusterSelector(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &nomosv1.ClusterSelector{
			TypeMeta: toTypeMeta(kinds.ClusterSelector()),
		},
	}
}

// Cluster returns a K8S Cluster resource at the specified path.
func Cluster(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &clusterregistry.Cluster{
			TypeMeta: toTypeMeta(kinds.Cluster()),
		},
	}
}

// ClusterConfig returns a ClusterConfig.
func ClusterConfig() ast.FileObject {
	return ast.FileObject{
		Object: &nomosv1.ClusterConfig{
			TypeMeta: toTypeMeta(kinds.ClusterConfig()),
			ObjectMeta: v1.ObjectMeta{
				Name: nomosv1.ClusterConfigName,
			},
		},
	}
}

// NamespaceConfig returns a default NamespaceConfig.
func NamespaceConfig() ast.FileObject {
	return ast.FileObject{
		Object: &nomosv1.NamespaceConfig{
			TypeMeta: toTypeMeta(kinds.NamespaceConfig()),
		},
	}
}

// Repo returns a nomos Repo at the specified path.
func Repo(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &nomosv1.Repo{
			TypeMeta: toTypeMeta(kinds.Repo()),
			ObjectMeta: v1.ObjectMeta{
				Name: "repo",
			},
			Spec: nomosv1.RepoSpec{
				Version: system.AllowedRepoVersion,
			},
		},
	}
}

// HierarchyConfig returns an empty HierarchyConfig at the specified path.
func HierarchyConfig(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &nomosv1.HierarchyConfig{
			TypeMeta: toTypeMeta(kinds.HierarchyConfig()),
		},
	}
}

// Sync returns a nomos Sync at the specified path.
func Sync(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &nomosv1.Sync{
			TypeMeta: toTypeMeta(kinds.Sync()),
		},
	}
}

// PersistentVolume returns a PersistentVolume Object.
func PersistentVolume() ast.FileObject {
	return ast.FileObject{
		Object: &corev1.PersistentVolume{
			TypeMeta: toTypeMeta(kinds.PersistentVolume()),
		},
	}
}

// Deployment returns a default Deployment object.
func Deployment(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &appsv1.Deployment{
			TypeMeta: toTypeMeta(kinds.Deployment()),
		},
	}
}

// ReplicaSet returns a default ReplicaSet object.
func ReplicaSet(path string) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &appsv1.ReplicaSet{
			TypeMeta: toTypeMeta(kinds.ReplicaSet()),
		},
	}
}

func toTypeMeta(gvk schema.GroupVersionKind) v1.TypeMeta {
	return v1.TypeMeta{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
	}
}
