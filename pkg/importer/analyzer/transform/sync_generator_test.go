package transform

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"

	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
)

var syncGeneratorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewSyncGenerator()
	},
	Options: func() []cmp.Option {
		return []cmp.Option{
			cmp.AllowUnexported(ast.FileObject{}),
		}
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name: "basic",
			Input: &ast.Root{
				ClusterObjects: vt.Helper.AcmeCluster(),
				Tree:           vt.Helper.AcmeTree(),
			},
			ExpectOutput: &ast.Root{
				SystemObjects: vt.SystemObjectSets(
					v1.NewSync(kinds.ClusterRole().GroupKind()),
					v1.NewSync(kinds.ClusterRoleBinding().GroupKind()),
					v1.NewSync(kinds.PodSecurityPolicy().GroupKind()),
					v1.NewSync(kinds.ResourceQuota().GroupKind()),
					v1.NewSync(kinds.Role().GroupKind()),
					v1.NewSync(kinds.RoleBinding().GroupKind()),
				),
				ClusterObjects: vt.Helper.AcmeCluster(),
				Tree:           vt.Helper.AcmeTree(),
			},
		},
	},
}

func TestSyncGenerator(t *testing.T) {
	t.Run("tests", syncGeneratorTestcases.Run)
}
