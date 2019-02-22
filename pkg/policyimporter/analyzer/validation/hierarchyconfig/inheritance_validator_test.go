package hierarchyconfig

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/object"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func HierarchyConfigResource(gvk schema.GroupVersionKind, mode v1.HierarchyModeType) object.BuildOpt {
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
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeDefault)),
		),
		asttest.Fail("inheritance rolebinding quota error",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance rolebinding inherit",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance rolebinding none",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeNone)),
		),
		asttest.Pass("inheritance resourcequota default",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeDefault)),
		),
		asttest.Pass("inheritance resourcequota quota error",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance resourcequota inherit",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance resourcequota none",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeNone)),
		))

	test.RunAll(t)
}
