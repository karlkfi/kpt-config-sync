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

func deprecatedDeployment() *ast.FileObject {
	o := ast.NewFileObject(
		&v1alpha1.ClusterRole{
			TypeMeta: v1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       kinds.Deployment().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "deployment",
			},
		},
		cmpath.FromSlash("namespaces/foo/deployment.yaml"),
	)
	return &o
}

func customResourceDefinition() *ast.FileObject {
	o := ast.NewFileObject(
		&v1alpha1.ClusterRole{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.CustomResourceDefinition().GroupVersion().String(),
				Kind:       kinds.CustomResourceDefinition().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "customResourceDefinition",
			},
		},
		cmpath.FromSlash("cluster/crd.yaml"),
	)
	return &o
}

func clusterRole() *ast.FileObject {
	o := ast.NewFileObject(
		&v1alpha1.ClusterRole{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.ClusterRole().GroupVersion().String(),
				Kind:       kinds.ClusterRole().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "clusterRole",
			},
		},
		cmpath.FromSlash("cluster/cr.yaml"),
	)
	return &o
}

func role() *ast.FileObject {
	o := ast.NewFileObject(
		&v1alpha1.Role{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.Role().GroupVersion().String(),
				Kind:       kinds.Role().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "role",
			},
		},
		cmpath.FromSlash("namespaces/foo/role.yaml"),
	)
	return &o
}

func replicaSet() *ast.FileObject {
	o := ast.NewFileObject(
		&appsv1.ReplicaSet{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.ReplicaSet().GroupVersion().String(),
				Kind:       kinds.ReplicaSet().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "replicaSet",
			},
		},
		cmpath.FromSlash("namespaces/foo/replicaset.yaml"),
	)
	return &o
}

func resourceQuota() *ast.FileObject {
	o := ast.NewFileObject(
		&corev1.ResourceQuota{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.ResourceQuota().GroupVersion().String(),
				Kind:       kinds.ResourceQuota().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "resourceQuota",
			},
		},
		cmpath.FromSlash("namespaces/foo/role.yaml"),
	)
	return &o
}

func namespace(path cmpath.Path) *ast.FileObject {
	o := ast.NewFileObject(
		&corev1.Namespace{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.Namespace().GroupVersion().String(),
				Kind:       kinds.Namespace().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: path.Dir().Base(),
			},
		},
		path,
	)
	return &o
}

func hierarchyConfig() *ast.FileObject {
	o := ast.NewFileObject(
		&corev1.Namespace{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.HierarchyConfig().GroupVersion().String(),
				Kind:       kinds.HierarchyConfig().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "hierarchyconfig",
			},
		},
		cmpath.FromSlash("system/hc.yaml"),
	)
	return &o
}
