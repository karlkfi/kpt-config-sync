package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestKindValidator(t *testing.T) {
	v := func() ast.Visitor {
		return NewHierarchyConfigKindValidator()
	}
	test := asttest.Validator(v,
		UnsupportedResourceInHierarchyConfigErrorCode,

		asttest.Pass("RoleBinding supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.RoleBinding())),
		),
		asttest.Pass("CRD supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.CustomResourceDefinitionV1Beta1())),
		),
		asttest.Fail("Namespace not supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.Namespace())),
		),
		asttest.Fail("omitting kind not supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, fake.GVK(kinds.RoleBinding(), fake.Kind("")))),
		),
		asttest.Pass("omitting group supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, fake.GVK(kinds.RoleBinding(), fake.Group("")))),
		),
		asttest.Fail("configmanagement.gke.io group not supported",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.Sync()))),
	)

	test.RunAll(t)
}
