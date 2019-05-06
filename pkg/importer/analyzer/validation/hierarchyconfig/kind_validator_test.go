package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestKindValidator(t *testing.T) {
	v := func() ast.Visitor {
		return NewHierarchyConfigKindValidator(false)
	}
	test := asttest.Validator(v,
		vet.UnsupportedResourceInHierarchyConfigErrorCode,

		asttest.Pass("RoleBinding supported",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeDefault)),
		),
		asttest.Fail("CRD not supported",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.CustomResourceDefinition(), v1.HierarchyModeDefault)),
		),
		asttest.Fail("Namespace not supported",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.Namespace(), v1.HierarchyModeDefault)),
		),
		asttest.Fail("omitting kind not supported",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(fake.GVK(kinds.RoleBinding(), fake.Kind("")), v1.HierarchyModeDefault)),
		),
		asttest.Pass("omitting group supported",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(fake.GVK(kinds.RoleBinding(), fake.Group("")), v1.HierarchyModeDefault)),
		),
		asttest.Fail("configmanagement.gke.io group not supported",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.Sync(), v1.HierarchyModeDefault))),
	)

	test.RunAll(t)
}
