package fake

import (
	nomos "github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		Object: &nomos.NamespaceSelector{
			TypeMeta: toTypeMeta(kinds.NamespaceSelector()),
		},
	}
}

// Role returns an RBAC Role at the specified path.
func Role(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &v1alpha1.Role{
			TypeMeta: toTypeMeta(kinds.Role()),
		},
	}
}

// RoleBinding returns an RBAC RoleBinding at the specified path.
func RoleBinding(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object: &v1alpha1.RoleBinding{
			TypeMeta: toTypeMeta(kinds.Role()),
		},
	}
}

func toTypeMeta(gvk schema.GroupVersionKind) v1.TypeMeta {
	return v1.TypeMeta{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
	}
}
