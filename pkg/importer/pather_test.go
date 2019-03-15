package importer

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/testing/apiresource"
	"github.com/google/nomos/pkg/testing/object"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func withoutPath() object.BuildOpt {
	return func(o *ast.FileObject) {
		o.Path = nomospath.Path{}
	}
}

func TestPatherSingleObject(t *testing.T) {
	testCases := []struct {
		name     string
		object   ast.FileObject
		expected nomospath.Path
	}{
		{
			name: "does nothing if object is nil",
		},
		{
			name:     "Repo",
			object:   object.Build(kinds.Repo(), withoutPath()),
			expected: nomospath.FromSlash(repo.SystemDir).Join(repoBasePath),
		},
		{
			name:     "HierarchyConfig",
			object:   object.Build(kinds.HierarchyConfig(), object.Name("hc"), withoutPath()),
			expected: nomospath.FromSlash(repo.SystemDir).Join(strings.ToLower(kinds.HierarchyConfig().Kind) + "_hc.yaml"),
		},
		{
			name:     "Clusters",
			object:   object.Build(kinds.Cluster(), object.Name("us-east-1"), withoutPath()),
			expected: nomospath.FromSlash(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.Cluster().Kind) + "_us-east-1.yaml"),
		},
		{
			name:     "ClusterSelector",
			object:   object.Build(kinds.ClusterSelector(), object.Name("cs"), withoutPath()),
			expected: nomospath.FromSlash(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.ClusterSelector().Kind) + "_cs.yaml"),
		},
		{
			name:     "Namespace prod",
			object:   object.Build(kinds.Namespace(), object.Name("prod"), withoutPath()),
			expected: nomospath.FromSlash(repo.NamespacesDir).Join("prod").Join(namespaceBasePath),
		},
		{
			name:     "Namespace dev",
			object:   object.Build(kinds.Namespace(), object.Name("dev"), withoutPath()),
			expected: nomospath.FromSlash(repo.NamespacesDir).Join("dev").Join(namespaceBasePath),
		},
		{
			name:     "Namespaced kind",
			object:   object.Build(kinds.Role(), object.Namespace("dev"), object.Name("admin"), withoutPath()),
			expected: nomospath.FromSlash(repo.NamespacesDir).Join("dev").Join("role_admin.yaml"),
		},
		{
			name:     "Clusters kind",
			object:   object.Build(kinds.ClusterRole(), object.Name("admin"), withoutPath()),
			expected: nomospath.FromSlash(repo.ClusterDir).Join("clusterrole_admin.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewPather(apiresource.Roles(), apiresource.ClusterRoles())

			objects := []ast.FileObject{tc.object}
			p.AddPaths(objects)

			if diff := cmp.Diff(tc.expected, objects[0].Path); diff != "" {
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
		expected nomospath.Path
	}{
		{
			name: "kind/name conflict",
			object: object.Build(object.GVK(kinds.Role(), object.Group("foo")),
				object.Name("admin"), object.Namespace("dev")),
			expected: nomospath.FromSlash(repo.NamespacesDir).Join("dev").Join("role_foo_admin.yaml"),
		},
		{
			name: "different namespace",
			object: object.Build(object.GVK(kinds.Role(), object.Group("foo")),
				object.Name("admin"), object.Namespace("prod")),
			expected: nomospath.FromSlash(repo.NamespacesDir).Join("prod").Join("role_admin.yaml"),
		},
		{
			name: "different kind",
			object: object.Build(object.GVK(kinds.RoleBinding(), object.Group("foo")),
				object.Name("admin"), object.Namespace("dev")),
			expected: nomospath.FromSlash(repo.NamespacesDir).Join("dev").Join("rolebinding_admin.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := Pather{
				namespaced: map[schema.GroupVersionKind]bool{
					object.GVK(kinds.Role(), object.Group("bar")):        true,
					object.GVK(kinds.Role(), object.Group("foo")):        true,
					object.GVK(kinds.RoleBinding(), object.Group("foo")): true,
				},
			}

			objects := []ast.FileObject{tc.object, other}
			p.AddPaths(objects)

			if diff := cmp.Diff(tc.expected, objects[0].Path); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
