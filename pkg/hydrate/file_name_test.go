package hydrate

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func cluster(name string) core.MetaMutator {
	return core.Annotation(v1.ClusterNameAnnotationKey, name)
}

func TestToFileObjects(t *testing.T) {
	testCases := []struct {
		name     string
		objects  []core.Object
		expected []ast.FileObject
	}{
		{
			name: "nil returns empty",
		},
		{
			name: "namespaced role works",
			objects: []core.Object{
				fake.RoleObject(core.Name("alice"), core.Namespace("prod"), cluster("na-1")),
			},
			expected: []ast.FileObject{
				fake.FileObject(fake.RoleObject(core.Name("alice"), core.Namespace("prod"), cluster("na-1")), "na-1/prod/role_alice.yaml"),
			},
		},
		{
			name: "non-namespaced clusterrolebinding works",
			objects: []core.Object{
				fake.ClusterRoleBindingObject(core.Name("alice"), cluster("eu-2")),
			},
			expected: []ast.FileObject{
				fake.FileObject(fake.ClusterRoleBindingObject(core.Name("alice"), cluster("eu-2")), "eu-2/clusterrolebinding_alice.yaml"),
			},
		},
		{
			name: "conflict resolved",
			objects: []core.Object{
				fake.UnstructuredObject(schema.GroupVersionKind{
					Group: "rbac",
					Kind:  "ClusterRole",
				}, core.Name("alice")),
				fake.UnstructuredObject(schema.GroupVersionKind{
					Group: "oauth",
					Kind:  "ClusterRole",
				}, core.Name("alice")),
			},
			expected: []ast.FileObject{
				fake.UnstructuredAtPath(schema.GroupVersionKind{
					Group: "rbac",
					Kind:  "ClusterRole",
				}, "defaultcluster/clusterrole.rbac_alice.yaml", core.Name("alice")),
				fake.UnstructuredAtPath(schema.GroupVersionKind{
					Group: "oauth",
					Kind:  "ClusterRole",
				}, "defaultcluster/clusterrole.oauth_alice.yaml", core.Name("alice")),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := ToFileObjects("yaml", true, tc.objects...)
			if diff := cmp.Diff(tc.expected, actual, cmpopts.EquateEmpty(), cmpopts.SortSlices(func(x, y ast.FileObject) bool {
				return x.SlashPath() < y.SlashPath()
			})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
