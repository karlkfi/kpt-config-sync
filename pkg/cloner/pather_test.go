package cloner

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/testing/object"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func withoutPath() object.BuildOpt {
	return func(o *ast.FileObject) {
		o.Relative = nomospath.Relative{}
	}
}

func TestPatherSingleObject(t *testing.T) {
	testCases := []struct {
		name     string
		object   ast.FileObject
		expected nomospath.Relative
	}{
		{
			name: "does nothing if object is nil",
		},
		{
			name:     "Repo",
			object:   object.Build(kinds.Repo(), withoutPath()),
			expected: nomospath.NewRelative(repo.SystemDir).Join(repoBasePath),
		},
		{
			name:     "HierarchyConfig",
			object:   object.Build(kinds.HierarchyConfig(), object.Name("hc"), withoutPath()),
			expected: nomospath.NewRelative(repo.SystemDir).Join(strings.ToLower(kinds.HierarchyConfig().Kind) + "_hc.yaml"),
		},
		{
			name:     "Cluster",
			object:   object.Build(kinds.Cluster(), object.Name("us-east-1"), withoutPath()),
			expected: nomospath.NewRelative(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.Cluster().Kind) + "_us-east-1.yaml"),
		},
		{
			name:     "ClusterSelector",
			object:   object.Build(kinds.ClusterSelector(), object.Name("cs"), withoutPath()),
			expected: nomospath.NewRelative(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.ClusterSelector().Kind) + "_cs.yaml"),
		},
		{
			name:     "Namespace prod",
			object:   object.Build(kinds.Namespace(), object.Name("prod"), withoutPath()),
			expected: nomospath.NewRelative(repo.NamespacesDir).Join("prod").Join(namespaceBasePath),
		},
		{
			name:     "Namespace dev",
			object:   object.Build(kinds.Namespace(), object.Name("dev"), withoutPath()),
			expected: nomospath.NewRelative(repo.NamespacesDir).Join("dev").Join(namespaceBasePath),
		},
		{
			name:     "Namespaced kind",
			object:   object.Build(kinds.Role(), object.Namespace("dev"), object.Name("admin"), withoutPath()),
			expected: nomospath.NewRelative(repo.NamespacesDir).Join("dev").Join("role_admin.yaml"),
		},
		{
			name:     "Cluster kind",
			object:   object.Build(kinds.ClusterRole(), object.Name("admin"), withoutPath()),
			expected: nomospath.NewRelative(repo.ClusterDir).Join("clusterrole_admin.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := pather{
				namespaced: map[schema.GroupVersionKind]bool{
					kinds.Role():        true,
					kinds.ClusterRole(): false,
				},
			}

			objects := []ast.FileObject{tc.object}
			p.addPaths(objects)

			if diff := cmp.Diff(tc.expected, objects[0].Relative); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestPatherMultipleObjects(t *testing.T) {
	other := object.Build(object.GVK(kinds.Role(), object.Group("bar")), object.Name("admin"), object.Namespace("dev"), withoutPath())

	testCases := []struct {
		name     string
		object   ast.FileObject
		expected nomospath.Relative
	}{
		{
			name: "kind/name conflict",
			object: object.Build(object.GVK(kinds.Role(), object.Group("foo")),
				object.Name("admin"), object.Namespace("dev")),
			expected: nomospath.NewRelative(repo.NamespacesDir).Join("dev").Join("role_foo_admin.yaml"),
		},
		{
			name: "different namespace",
			object: object.Build(object.GVK(kinds.Role(), object.Group("foo")),
				object.Name("admin"), object.Namespace("prod")),
			expected: nomospath.NewRelative(repo.NamespacesDir).Join("prod").Join("role_admin.yaml"),
		},
		{
			name: "different kind",
			object: object.Build(object.GVK(kinds.RoleBinding(), object.Group("foo")),
				object.Name("admin"), object.Namespace("dev")),
			expected: nomospath.NewRelative(repo.NamespacesDir).Join("dev").Join("rolebinding_admin.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := pather{
				namespaced: map[schema.GroupVersionKind]bool{
					object.GVK(kinds.Role(), object.Group("bar")):        true,
					object.GVK(kinds.Role(), object.Group("foo")):        true,
					object.GVK(kinds.RoleBinding(), object.Group("foo")): true,
				},
			}

			objects := []ast.FileObject{tc.object, other}
			p.addPaths(objects)

			if diff := cmp.Diff(tc.expected, objects[0].Relative); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
