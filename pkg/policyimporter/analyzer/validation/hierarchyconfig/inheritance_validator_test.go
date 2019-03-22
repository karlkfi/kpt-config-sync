package hierarchyconfig

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func HierarchyConfigResource(gvk schema.GroupVersionKind, mode v1.HierarchyModeType) object.Mutator {
	return func(o *ast.FileObject) {
		o.Object.(*v1.HierarchyConfig).Spec.Resources = append(o.Object.(*v1.HierarchyConfig).Spec.Resources,
			v1.HierarchyConfigResource{
				Group:         gvk.Group,
				Kinds:         []string{gvk.Kind},
				HierarchyMode: mode,
			})
	}
}

func TestInheritanceValidator(t *testing.T) {
	test := asttest.Validator(NewInheritanceValidator,
		vet.IllegalHierarchyModeErrorCode,

		asttest.Pass("inheritance rolebinding default",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeDefault)),
		),
		asttest.Fail("inheritance rolebinding quota error",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance rolebinding inherit",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance rolebinding none",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeNone)),
		),
		asttest.Pass("inheritance resourcequota default",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeDefault)),
		),
		asttest.Pass("inheritance resourcequota quota error",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance resourcequota inherit",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance resourcequota none",
			fake.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeNone)),
		))

	test.RunAll(t)
}
