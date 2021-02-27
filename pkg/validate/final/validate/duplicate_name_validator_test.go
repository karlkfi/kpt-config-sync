package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func customResource(group, kind, name, namespace string) ast.FileObject {
	gvk := schema.GroupVersionKind{
		Group:   group,
		Version: "v1",
		Kind:    kind,
	}
	return fake.Unstructured(gvk, core.Name(name), core.Namespace(namespace))
}

func TestDuplicateNames(t *testing.T) {
	testCases := []struct {
		name     string
		objs     []ast.FileObject
		wantErrs status.MultiError
	}{
		{
			name: "Two objects with different names pass",
			objs: []ast.FileObject{
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
				fake.Role(core.Name("bob"), core.Namespace("shipping")),
			},
		},
		{
			name: "Two objects with different namespaces pass",
			objs: []ast.FileObject{
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
				fake.Role(core.Name("alice"), core.Namespace("production")),
			},
		},
		{
			name: "Two objects with different kinds pass",
			objs: []ast.FileObject{
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
				fake.RoleBinding(core.Name("alice"), core.Namespace("shipping")),
			},
		},
		{
			name: "Two objects with different groups pass",
			objs: []ast.FileObject{
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
				customResource("acme", "Role", "alice", "shipping"),
			},
		},
		{
			name: "Two duplicate namespaced objects fail",
			objs: []ast.FileObject{
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
				fake.Role(core.Name("alice"), core.Namespace("shipping")),
			},
			wantErrs: nonhierarchical.NamespaceMetadataNameCollisionError(
				kinds.Role().GroupKind(), "shipping", "alice", fake.Role()),
		},
		{
			name: "Two duplicate cluster-scoped objects fail",
			objs: []ast.FileObject{
				fake.ClusterRole(core.Name("alice")),
				fake.ClusterRole(core.Name("alice")),
			},
			wantErrs: nonhierarchical.ClusterMetadataNameCollisionError(
				kinds.ClusterRole().GroupKind(), "alice", fake.ClusterRole()),
		},
		{
			name: "Two duplicate namespaces fail",
			objs: []ast.FileObject{
				fake.Namespace("namespaces/hello"),
				fake.Namespace("namespaces/hello"),
			},
			wantErrs: nonhierarchical.NamespaceCollisionError(
				"hello", fake.Namespace("hamespaces/hello")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := DuplicateNames(tc.objs)
			if !errors.Is(errs, tc.wantErrs) {
				t.Errorf("got DuplicateNames() error %v, want %v", errs, tc.wantErrs)
			}
		})
	}
}
