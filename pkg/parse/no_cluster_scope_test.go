package parse

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestClusterScopeValidator(t *testing.T) {
	var scoper discovery.Scoper = map[schema.GroupKind]discovery.IsNamespaced{
		kinds.Role().GroupKind():        true,
		kinds.ClusterRole().GroupKind(): false,
		kinds.Namespace().GroupKind():   false,
	}

	testCases := []struct {
		name     string
		obj      core.Object
		wantErrs []string
	}{
		{
			name: "Role",
			obj:  fake.RoleObject(),
		},
		{
			name:     "ClusterRole",
			obj:      fake.ClusterRoleObject(),
			wantErrs: []string{BadScopeErrCode},
		},
		{
			name:     "Namespace",
			obj:      fake.NamespaceObject("foo"),
			wantErrs: []string{BadScopeErrCode},
		},
		{
			name: "Unknown type",
			obj: fake.UnstructuredObject(schema.GroupVersionKind{
				Group: "anvils.com",
				Kind:  "Anvil",
			}),
			wantErrs: []string{discovery.UnknownKindErrorCode},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := noClusterScopeValidator(scoper)
			got := v.Validate([]ast.FileObject{{Object: tc.obj}})

			vettesting.ExpectErrors(tc.wantErrs, got, t)
		})
	}
}
