package transform

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"

	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

var syncGeneratorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return NewSyncGenerator()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name: "basic",
			Input: &ast.Root{
				System:  &ast.System{},
				Cluster: vt.Helper.AcmeCluster(),
				Tree:    vt.Helper.AcmeTree(),
			},
			ExpectOutput: &ast.Root{
				System: &ast.System{
					Objects: vt.SystemObjectSets(
						v1alpha1.NewSync("rbac.authorization.k8s.io", "ClusterRole"),
						v1alpha1.NewSync("rbac.authorization.k8s.io", "ClusterRoleBinding"),
						v1alpha1.NewSync("extensions", "PodSecurityPolicy"),
						v1alpha1.NewSync("", "ResourceQuota"),
						v1alpha1.NewSync("rbac.authorization.k8s.io", "Role"),
						v1alpha1.NewSync("rbac.authorization.k8s.io", "RoleBinding"),
					),
				},
				Cluster: vt.Helper.AcmeCluster(),
				Tree:    vt.Helper.AcmeTree(),
			},
		},
	},
}

func TestSyncGenerator(t *testing.T) {
	t.Run("tests", syncGeneratorTestcases.Run)
}
