package vet

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func customResourceDefinition() *ast.FileObject {
	return &ast.FileObject{
		Path: cmpath.FromSlash("cluster/crd.yaml"),
		Object: &v1alpha1.ClusterRole{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.CustomResourceDefinition().GroupVersion().String(),
				Kind:       kinds.CustomResourceDefinition().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "customResourceDefinition",
			},
		},
	}
}

func clusterRole() *ast.FileObject {
	return &ast.FileObject{
		Path: cmpath.FromSlash("cluster/cr.yaml"),
		Object: &v1alpha1.ClusterRole{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.ClusterRole().GroupVersion().String(),
				Kind:       kinds.ClusterRole().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "clusterRole",
			},
		},
	}
}

func role() *ast.FileObject {
	return &ast.FileObject{
		Path: cmpath.FromSlash("namespaces/foo/role.yaml"),
		Object: &v1alpha1.Role{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.Role().GroupVersion().String(),
				Kind:       kinds.Role().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "role",
			},
		},
	}
}

func replicaSet() *ast.FileObject {
	return &ast.FileObject{
		Path: cmpath.FromSlash("namespaces/foo/replicaset.yaml"),
		Object: &appsv1.ReplicaSet{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.ReplicaSet().GroupVersion().String(),
				Kind:       kinds.ReplicaSet().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "replicaSet",
			},
		},
	}
}

func resourceQuota() *ast.FileObject {
	return &ast.FileObject{
		Path: cmpath.FromSlash("namespaces/foo/role.yaml"),
		Object: &corev1.ResourceQuota{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.ResourceQuota().GroupVersion().String(),
				Kind:       kinds.ResourceQuota().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "resourceQuota",
			},
		},
	}
}

func namespace(path cmpath.Path) *ast.FileObject {
	return &ast.FileObject{
		Path: path,
		Object: &corev1.Namespace{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.Namespace().GroupVersion().String(),
				Kind:       kinds.Namespace().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: path.Dir().Base(),
			},
		},
	}
}

func hierarchyConfig() *ast.FileObject {
	return &ast.FileObject{
		Path: cmpath.FromSlash("system/hc.yaml"),
		Object: &corev1.Namespace{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.HierarchyConfig().GroupVersion().String(),
				Kind:       kinds.HierarchyConfig().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "hierarchyconfig",
			},
		},
	}
}
