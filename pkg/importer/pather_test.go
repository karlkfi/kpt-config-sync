package importer

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/apiresource"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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
			name:     configmanagement.RepoKind,
			object:   fake.Repo(),
			expected: cmpath.FromSlash(repo.SystemDir).Join(repoBasePath),
		},
		{
			name:     configmanagement.HierarchyConfigKind,
			object:   fake.HierarchyConfig(core.Name("hc")),
			expected: cmpath.FromSlash(repo.SystemDir).Join(strings.ToLower(kinds.HierarchyConfig().Kind) + "_hc.yaml"),
		},
		{
			name:     "Clusters",
			object:   fake.Cluster(core.Name("us-east-1")),
			expected: cmpath.FromSlash(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.Cluster().Kind) + "_us-east-1.yaml"),
		},
		{
			name:     configmanagement.ClusterSelectorKind,
			object:   fake.ClusterSelector(core.Name("cs")),
			expected: cmpath.FromSlash(repo.ClusterRegistryDir).Join(strings.ToLower(kinds.ClusterSelector().Kind) + "_cs.yaml"),
		},
		{
			name:     "Namespace prod",
			object:   fake.Namespace("namespaces/prod"),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("prod").Join(namespaceBasePath),
		},
		{
			name:     "Namespace dev",
			object:   fake.Namespace("namespaces/dev"),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("dev").Join(namespaceBasePath),
		},
		{
			name:     "Namespaced kind",
			object:   fake.Role(core.Namespace("dev"), core.Name("admin")),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("dev").Join("role_admin.yaml"),
		},
		{
			name:     "Clusters kind",
			object:   fake.ClusterRole(core.Name("admin")),
			expected: cmpath.FromSlash(repo.ClusterDir).Join("clusterrole_admin.yaml"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.object.Path = cmpath.Path{} // unset path.
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
	other := fake.Unstructured(fake.GVK(kinds.Role(), fake.Group("bar")), core.Name("admin"), core.Namespace("dev"))

	testCases := []struct {
		name     string
		object   ast.FileObject
		expected cmpath.Path
	}{
		{
			name: "kind/name conflict",
			object: fake.Unstructured(fake.GVK(kinds.Role(), fake.Group("foo")),
				core.Name("admin"), core.Namespace("dev")),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("dev").Join("role_foo_admin.yaml"),
		},
		{
			name: "different namespace",
			object: fake.Unstructured(fake.GVK(kinds.Role(), fake.Group("foo")),
				core.Name("admin"), core.Namespace("prod")),
			expected: cmpath.FromSlash(repo.NamespacesDir).Join("prod").Join("role_admin.yaml"),
		},
		{
			name: "different kind",
			object: fake.Unstructured(fake.GVK(kinds.RoleBinding(), fake.Group("foo")),
				core.Name("admin"), core.Namespace("dev")),
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
