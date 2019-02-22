package hierarchyconfig

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/object"
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func toAPIInfo(known ...schema.GroupVersionKind) (*discovery.APIInfo, error) {
	resources := make([]*metav1.APIResourceList, len(known))

	for i, gvk := range known {
		resources[i] = &metav1.APIResourceList{
			GroupVersion: gvk.GroupVersion().String(),
			APIResources: []metav1.APIResource{{Kind: gvk.Kind}},
		}
	}

	return discovery.NewAPIInfo(resources)
}

// APIInfo adds an APIInfo to the AST.
func APIInfo(apiInfo *discovery.APIInfo) ast.BuildOpt {
	return func(root *ast.Root) error {
		if apiInfo == nil {
			return nil
		}
		discovery.AddAPIInfo(root, apiInfo)
		return nil
	}
}

func TestKnownResourceValidator(t *testing.T) {
	apiInfo, err := toAPIInfo(kinds.RoleBinding())
	if err != nil {
		t.Fatalf("unexpected error forming APIInfo: %v", err)
	}

	test := asttest.Validator(NewKnownResourceValidator,
		vet.UnknownResourceInHierarchyConfigErrorCode,

		asttest.Fail("ResourceQuota throws error if not known",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.ResourceQuota(), v1.HierarchyModeDefault)),
		),
		asttest.Pass("RoleBinding valid if known",
			object.Build(kinds.HierarchyConfig(),
				HierarchyConfigResource(kinds.RoleBinding(), v1.HierarchyModeDefault)),
		),
	).With(APIInfo(apiInfo))

	test.RunAll(t)
}
