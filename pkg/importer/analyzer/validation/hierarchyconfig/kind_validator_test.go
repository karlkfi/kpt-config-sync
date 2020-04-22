package hierarchyconfig

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestKindValidator(t *testing.T) {
	v := func() ast.Visitor {
		return NewHierarchyConfigKindValidator()
	}
	asttest.Validator(t, v,
		UnsupportedResourceInHierarchyConfigErrorCode,

		asttest.Pass("RoleBinding supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.RoleBinding())),
		),
		asttest.Pass("v1Beta1 CRD supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.CustomResourceDefinitionV1Beta1())),
		),
		asttest.Pass("v1 CRD supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.CustomResourceDefinitionV1())),
		),
		asttest.Fail("Namespace not supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.Namespace())),
		),
		asttest.Fail("omitting kind not supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.RoleBinding().GroupVersion().WithKind(""))),
		),
		asttest.Pass("omitting group supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, schema.GroupVersionKind{Version: "v1", Kind: "Role"})),
		),
		asttest.Fail("configmanagement.gke.io group not supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.Sync()))),
	)
}
