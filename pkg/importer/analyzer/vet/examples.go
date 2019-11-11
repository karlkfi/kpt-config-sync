package vet

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
)

func deprecatedDeployment() *ast.FileObject {
	o := ast.NewFileObject(
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       kinds.Deployment().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "deployment",
			},
		},
		cmpath.FromSlash("namespaces/foo/deployment.yaml"),
	)
	return &o
}

func customResourceDefinition() *ast.FileObject {
	o := ast.NewFileObject(
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kinds.CustomResourceDefinition().GroupVersion().String(),
				Kind:       kinds.CustomResourceDefinition().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "customResourceDefinition",
			},
		},
		cmpath.FromSlash("cluster/crd.yaml"),
	)
	return &o
}

func clusterRole() *ast.FileObject {
	o := ast.NewFileObject(
		&rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kinds.ClusterRole().GroupVersion().String(),
				Kind:       kinds.ClusterRole().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "clusterRole",
			},
		},
		cmpath.FromSlash("cluster/cr.yaml"),
	)
	return &o
}

func role() *ast.FileObject {
	o := ast.NewFileObject(
		&rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kinds.Role().GroupVersion().String(),
				Kind:       kinds.Role().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "role",
			},
		},
		cmpath.FromSlash("namespaces/foo/role.yaml"),
	)
	return &o
}

func resourceQuota() *ast.FileObject {
	o := ast.NewFileObject(
		&corev1.ResourceQuota{
			TypeMeta: metav1.TypeMeta{
				APIVersion: kinds.ResourceQuota().GroupVersion().String(),
				Kind:       kinds.ResourceQuota().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
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
			TypeMeta: metav1.TypeMeta{
				APIVersion: kinds.Namespace().GroupVersion().String(),
				Kind:       kinds.Namespace().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
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
			TypeMeta: metav1.TypeMeta{
				APIVersion: kinds.HierarchyConfig().GroupVersion().String(),
				Kind:       kinds.HierarchyConfig().Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "hierarchyconfig",
			},
		},
		cmpath.FromSlash("system/hc.yaml"),
	)
	return &o
}
