package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/node"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func namespace(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object:   &corev1.Namespace{TypeMeta: metav1.TypeMeta{APIVersion: kinds.Namespace().GroupVersion().String(), Kind: kinds.Namespace().Kind}},
	}
}

func role(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object:   &rbacv1.Role{TypeMeta: metav1.TypeMeta{APIVersion: kinds.Role().GroupVersion().String(), Kind: kinds.Role().Kind}},
	}
}

func namespaceSelector(path string) ast.FileObject {
	return ast.FileObject{
		Relative: nomospath.NewFakeRelative(path),
		Object:   &v1alpha1.NamespaceSelector{TypeMeta: metav1.TypeMeta{APIVersion: kinds.NamespaceSelector().GroupVersion().String(), Kind: kinds.NamespaceSelector().Kind}},
	}
}

func TestEphemeralResourceRemoverNamespace(t *testing.T) {
	testCases := []struct {
		name     string
		objects  []ast.FileObject
		expected *ast.Root
	}{
		{
			name:     "empty returns empty",
			expected: &ast.Root{},
		},
		{
			name:    "namespace returns empty",
			objects: []ast.FileObject{namespace("namespaces/bar/ns.yaml")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewFakeRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewFakeRelative("namespaces/bar"),
							Type:     node.Namespace,
						},
					},
				},
			},
		},
		{
			name:    "namespaceselector returns empty",
			objects: []ast.FileObject{namespaceSelector("namespaces/ns.yaml")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewFakeRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Selectors: map[string]*v1alpha1.NamespaceSelector{
						"": namespaceSelector("namespaces/ns.yaml").Object.(*v1alpha1.NamespaceSelector),
					},
				},
			},
		},
		{
			name:     "keeps non-ephemeral",
			objects:  []ast.FileObject{role("namespaces/bar/role.yaml")},
			expected: treetesting.BuildTree(role("namespaces/bar/role.yaml")),
		},
		{
			name:    "only non-ephemeral",
			objects: []ast.FileObject{namespace("namespaces/bar/ns.yaml"), role("namespaces/bar/role.yaml")},
			expected: &ast.Root{
				Tree: &ast.TreeNode{
					Relative: nomospath.NewFakeRelative("namespaces"),
					Type:     node.AbstractNamespace,
					Children: []*ast.TreeNode{
						{
							Relative: nomospath.NewFakeRelative("namespaces/bar"),
							Type:     node.Namespace,
							Objects:  []*ast.NamespaceObject{{FileObject: role("namespaces/bar/role.yaml")}},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := treetesting.BuildTree(tc.objects...)

			root.Accept(NewEphemeralResourceRemover())

			if diff := cmp.Diff(tc.expected, root); diff != "" {
				t.Fatalf("unexpected difference in trees\n\n%s", diff)
			}
		})
	}
}

func TestEphemeralResourceRemoverNonNamespace(t *testing.T) {
	testCases := []struct {
		name     string
		initial  *ast.Root
		expected *ast.Root
	}{
		{
			name: "role in System returns same",
			initial: &ast.Root{
				System: &ast.System{
					Objects: []*ast.SystemObject{{FileObject: role("system/role.yaml")}},
				},
			},
			expected: &ast.Root{
				System: &ast.System{
					Objects: []*ast.SystemObject{{FileObject: role("system/role.yaml")}},
				},
			},
		},
		{
			name: "role in ClusterRegistry returns same",
			initial: &ast.Root{
				ClusterRegistry: &ast.ClusterRegistry{
					Objects: []*ast.ClusterRegistryObject{{FileObject: role("clusterregistry/ns.yaml")}},
				},
			},
			expected: &ast.Root{
				ClusterRegistry: &ast.ClusterRegistry{
					Objects: []*ast.ClusterRegistryObject{{FileObject: role("clusterregistry/ns.yaml")}},
				},
			},
		},
		{
			name: "role in Cluster returns same",
			initial: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: []*ast.ClusterObject{{FileObject: role("cluster/ns.yaml")}},
				},
			},
			expected: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: []*ast.ClusterObject{{FileObject: role("cluster/ns.yaml")}},
				},
			},
		},
		{
			name: "namespace in System returns empty",
			initial: &ast.Root{
				System: &ast.System{
					Objects: []*ast.SystemObject{{FileObject: namespace("system/ns.yaml")}},
				},
			},
			expected: &ast.Root{
				System: &ast.System{},
			},
		},
		{
			name: "namespace in ClusterRegistry returns empty",
			initial: &ast.Root{
				ClusterRegistry: &ast.ClusterRegistry{
					Objects: []*ast.ClusterRegistryObject{{FileObject: namespace("clusterregistry/ns.yaml")}},
				},
			},
			expected: &ast.Root{
				ClusterRegistry: &ast.ClusterRegistry{},
			},
		},
		{
			name: "namespace in Cluster returns empty",
			initial: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: []*ast.ClusterObject{{FileObject: namespace("cluster/ns.yaml")}},
				},
			},
			expected: &ast.Root{
				Cluster: &ast.Cluster{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := tc.initial

			root.Accept(NewEphemeralResourceRemover())

			if diff := cmp.Diff(tc.expected, root); diff != "" {
				t.Fatalf("unexpected difference in trees\n\n%s", diff)
			}
		})
	}
}
