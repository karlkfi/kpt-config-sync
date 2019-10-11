package backend

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/resourcequota"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/namespaceconfig"
	"github.com/google/nomos/testing/testoutput"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var helper = vt.NewTestHelper()

func allConfigs(c, crd *v1.ClusterConfig, pns map[string]v1.NamespaceConfig) *namespaceconfig.AllConfigs {
	c.Spec.ImportTime = vt.ImportTime
	c.Spec.Token = vt.ImportToken

	crd.Spec.ImportTime = vt.ImportTime
	crd.Spec.Token = vt.ImportToken

	ap := &namespaceconfig.AllConfigs{
		ClusterConfig:    c,
		CRDClusterConfig: crd,
		NamespaceConfigs: map[string]v1.NamespaceConfig{},
		Syncs:            map[string]v1.Sync{},
		ImportToken:      vt.ImportToken,
	}
	for name, nc := range pns {
		nc.Spec.Token = vt.ImportToken
		nc.Spec.ImportTime = vt.ImportTime
		object.RemoveAnnotations(&nc, v1.SourcePathAnnotationKey)
		pns[name] = nc
	}
	ap.NamespaceConfigs = pns
	return ap
}

type OutputVisitorTestcase struct {
	name     string
	input    *ast.Root
	expected *namespaceconfig.AllConfigs
}

func (tc *OutputVisitorTestcase) Run(t *testing.T) {
	ov := NewOutputVisitor()
	tc.input.Accept(ov)
	actual := ov.AllConfigs()
	// LoadTime is hard to get right, so just set it to zero
	actual.LoadTime = metav1.NewTime(time.Time{})
	if diff := cmp.Diff(tc.expected, actual, resourcequota.ResourceQuantityEqual(), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("mismatch on expected vs actual: %s", diff)
	}
}

var outputVisitorTestCases = []OutputVisitorTestcase{
	{
		name: "empty cluster configs",
		input: &ast.Root{
			ImportToken: vt.ImportToken,
			LoadTime:    vt.ImportTime.Time,
		},
		expected: allConfigs(
			fake.ClusterConfigObject(),
			fake.CRDClusterConfigObject(),
			nil,
		),
	},
	{
		name:  "cluster configs",
		input: helper.ClusterConfigs(),
		expected: allConfigs(
			testoutput.ClusterConfig(
				helper.NomosAdminClusterRole(),
				helper.NomosAdminClusterRoleBinding(),
				helper.NomosPodSecurityPolicy(),
			),
			fake.CRDClusterConfigObject(),
			nil,
		),
	},
	{
		name:  "crd cluster configs",
		input: helper.CRDClusterConfig(),
		expected: allConfigs(
			testoutput.ClusterConfig(),
			testoutput.CRDClusterConfig(
				helper.CRD(),
			),
			nil,
		),
	},
	{
		name:  "namespace configs",
		input: helper.NamespaceConfigs(),
		expected: allConfigs(
			testoutput.ClusterConfig(),
			testoutput.CRDClusterConfig(),
			testoutput.NamespaceConfigs(
				testoutput.NamespaceConfig("", "frontend", object.MetaMutators(
					object.Annotation("has-waffles", "true"),
					object.Label("environment", "prod"),
				),
					helper.PodReaderRoleBinding(),
					helper.PodReaderRole(),
					helper.FrontendResourceQuota(),
				),
				testoutput.NamespaceConfig("", "frontend-test", object.MetaMutators(
					object.Annotation("has-waffles", "false"),
					object.Label("environment", "test"),
				),
					helper.DeploymentReaderRoleBinding(),
					helper.DeploymentReaderRole(),
				),
			),
		),
	},
}

func TestOutputVisitor(t *testing.T) {
	for _, tc := range outputVisitorTestCases {
		t.Run(tc.name, tc.Run)
	}
}
