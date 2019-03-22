package transform

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"

	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
)

var syncGeneratorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewSyncGenerator()
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
					v1.NewSync("rbac.authorization.k8s.io", "ClusterRole"),
					v1.NewSync("rbac.authorization.k8s.io", "ClusterRoleBinding"),
					v1.NewSync("policy", "PodSecurityPolicy"),
					v1.NewSync("", "ResourceQuota"),
					v1.NewSync("rbac.authorization.k8s.io", "Role"),
					v1.NewSync("rbac.authorization.k8s.io", "RoleBinding"),
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
