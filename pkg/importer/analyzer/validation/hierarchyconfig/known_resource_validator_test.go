package hierarchyconfig

import (
	"testing"

	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/util/discovery"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type apiInfoOption func([]*metav1.APIResourceList) []*metav1.APIResourceList

func apiResource(known schema.GroupVersionKind, namespaced bool) apiInfoOption {
	return func(list []*metav1.APIResourceList) []*metav1.APIResourceList {
		return append(list, &metav1.APIResourceList{
			GroupVersion: known.GroupVersion().String(),
			APIResources: []metav1.APIResource{
				{
					Kind:       known.Kind,
					Namespaced: namespaced,
				},
			},
		})
	}
}

func toAPIInfo(opts ...apiInfoOption) (discovery.Scoper, error) {
	var resources []*metav1.APIResourceList
	for _, o := range opts {
		resources = o(resources)
	}
	return discovery.NewAPIInfo(resources)
}

// APIInfo adds an APIInfo to the AST.
func APIInfo(apiInfo discovery.Scoper) ast.BuildOpt {
	return func(root *ast.Root) status.MultiError {
		if apiInfo == nil {
			return nil
		}
		discovery.AddScoper(root, apiInfo)
		return nil
	}
}

func TestKnownResourceValidatorUnknown(t *testing.T) {
	apiInfo, err := toAPIInfo(
		apiResource(kinds.RoleBinding(), true),
		apiResource(kinds.ClusterRoleBinding(), false),
	)
	if err != nil {
		t.Fatalf("unexpected error forming APIInfo: %v", err)
	}

	test := asttest.Validator(NewKnownResourceValidator,
		vet.UnknownResourceInHierarchyConfigErrorCode,

		asttest.Fail("ResourceQuota throws error if not known",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.ResourceQuota())),
		),
		asttest.Pass("RoleBinding valid if known",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.RoleBinding())),
		),
	).With(APIInfo(apiInfo))

	test.RunAll(t)
}

func TestKnownResourceValidatorScope(t *testing.T) {
	apiInfo, err := toAPIInfo(
		apiResource(kinds.RoleBinding(), true),
		apiResource(kinds.ClusterRoleBinding(), false),
	)
	if err != nil {
		t.Fatalf("unexpected error forming APIInfo: %v", err)
	}

	test := asttest.Validator(NewKnownResourceValidator,
		vet.ClusterScopedResourceInHierarchyConfigErrorCode,
		asttest.Fail("ClusterRoleBinding is cluster scoped",
			fake.HierarchyConfig(
				fake.HierarchyConfigKind(v1.HierarchyModeDefault, kinds.ClusterRoleBinding())),
		),
	).With(APIInfo(apiInfo))

	test.RunAll(t)
}
