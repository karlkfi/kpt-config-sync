package importer

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/cmpath"
	"github.com/google/nomos/pkg/testing/apiresource"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func withoutPath() object.Mutator {
	return func(o *ast.FileObject) {
		o.Path = cmpath.Path{}
	}
}

func TestPatherSingleObject(t *testing.T) {
	testCases := []struct {
		name     string
		object   ast.FileObject
		expected cmpath.Path
	}{
		{
			name: "does nothing if object is nil",
		},
		{
			name:     "Repo",
			object:   fake.Build(kinds.Repo(), withoutPath()),
			expected: cmpath.FromSlash(repo.SystemDir).Join(repoBasePath),
		},
		{
			name:     "HierarchyConfig",
			object:   fake.Build(kinds.HierarchyConfig(), object.Name("hc"), withoutPath()),
			expected: cmpath.FromSlash(repo.SystemDir).Join(strings.ToLower(kinds.HierarchyConfig().Kind) + "_hc.yaml"),
		},
		{
			name:     "Clusters",
			object:   fake.Build(kinds.Cluster(), object.Name("us-east-1"), withoutPath()),
			expected: cmpath.FromSlash(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.Cluster().Kind) + "_us-east-1.yaml"),
		},
		{
			name:     "ClusterSelector",
			object:   fake.Build(kinds.ClusterSelector(), object.Name("cs"), withoutPath()),
			expected: cmpath.FromSlash(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.ClusterSelector().Kind) + "_cs.yaml"),
		},
		{
			name:     "Namespace prod",
			object:   fake.Build(kinds.Namespace(), object.Name("prod"), withoutPath()),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("prod").Join(namespaceBasePath),
		},
		{
			name:     "Namespace dev",
			object:   fake.Build(kinds.Namespace(), object.Name("dev"), withoutPath()),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("dev").Join(namespaceBasePath),
		},
		{
			name:     "Namespaced kind",
			object:   fake.Build(kinds.Role(), object.Namespace("dev"), object.Name("admin"), withoutPath()),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("dev").Join("role_admin.yaml"),
		},
		{
			name:     "Clusters kind",
			object:   fake.Build(kinds.ClusterRole(), object.Name("admin"), withoutPath()),
			expected: cmpath.FromSlash(repo.ClusterDir).Join("clusterrole_admin.yaml"),
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
	other := fake.Build(fake.GVK(kinds.Role(), fake.Group("bar")), object.Name("admin"), object.Namespace("dev"), withoutPath())

	testCases := []struct {
		name     string
		object   ast.FileObject
		expected cmpath.Path
	}{
		{
			name: "kind/name conflict",
			object: fake.Build(fake.GVK(kinds.Role(), fake.Group("foo")),
				object.Name("admin"), object.Namespace("dev")),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("dev").Join("role_foo_admin.yaml"),
		},
		{
			name: "different namespace",
			object: fake.Build(fake.GVK(kinds.Role(), fake.Group("foo")),
				object.Name("admin"), object.Namespace("prod")),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("prod").Join("role_admin.yaml"),
		},
		{
			name: "different kind",
			object: fake.Build(fake.GVK(kinds.RoleBinding(), fake.Group("foo")),
				object.Name("admin"), object.Namespace("dev")),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("dev").Join("rolebinding_admin.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := Pather{
				namespaced: map[schema.GroupVersionKind]bool{
					fake.GVK(kinds.Role(), fake.Group("bar")):        true,
					fake.GVK(kinds.Role(), fake.Group("foo")):        true,
					fake.GVK(kinds.RoleBinding(), fake.Group("foo")): true,
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
