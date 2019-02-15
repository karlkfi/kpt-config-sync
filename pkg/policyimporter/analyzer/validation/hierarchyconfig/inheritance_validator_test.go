package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/object"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func HierarchyConfigResource(gvk schema.GroupVersionKind, mode v1alpha1.HierarchyModeType) object.BuildOpt {
	return func(o *ast.FileObject) {
		o.Object.(*v1alpha1.HierarchyConfig).Spec.Resources = append(o.Object.(*v1alpha1.HierarchyConfig).Spec.Resources,
			v1alpha1.HierarchyConfigResource{
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
				HierarchyConfigResource(kinds.RoleBinding(), v1alpha1.HierarchyModeDefault)),
		),
		asttest.Fail("inheritance rolebinding quota error",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1alpha1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance rolebinding inherit",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1alpha1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance rolebinding none",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1alpha1.HierarchyModeNone)),
		),
		asttest.Pass("inheritance resourcequota default",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1alpha1.HierarchyModeDefault)),
		),
		asttest.Pass("inheritance resourcequota quota error",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1alpha1.HierarchyModeHierarchicalQuota)),
		),
		asttest.Pass("inheritance resourcequota inherit",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1alpha1.HierarchyModeInherit)),
		),
		asttest.Pass("inheritance resourcequota none",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1alpha1.HierarchyModeNone)),
		))

	test.RunAll(t)
}
