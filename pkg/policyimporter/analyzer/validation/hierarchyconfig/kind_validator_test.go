package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/object"
)

func TestKindValidator(t *testing.T) {
	test := asttest.Validator(NewHierarchyConfigKindValidator,
		vet.UnsupportedResourceInHierarchyConfigErrorCode,

		asttest.Pass("RoleBinding supported",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1alpha1.HierarchyModeDefault)),
		),
		asttest.Fail("CRD not supported",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.CustomResourceDefinition(), v1alpha1.HierarchyModeDefault)),
		),
		asttest.Fail("Namespace not supported",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.Namespace(), v1alpha1.HierarchyModeDefault)),
		),
		asttest.Fail("omitting kind not supported",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(object.GVK(kinds.RoleBinding(), object.Kind("")), v1alpha1.HierarchyModeDefault)),
		),
		asttest.Pass("omitting group supported",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(object.GVK(kinds.RoleBinding(), object.Group("")), v1alpha1.HierarchyModeDefault)),
		),
		asttest.Fail("nomos.dev group not supported",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.Sync(), v1alpha1.HierarchyModeDefault))),
	)

	test.RunAll(t)
}
