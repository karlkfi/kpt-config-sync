package nonhierarchical_test

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	nht "github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical/nonhierarchicaltest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNewManagedAnnotationValidator(t *testing.T) {
	testCases := []nht.ValidatorTestCase{
		nht.Pass("no management annotation",
			fake.Role(),
		),
		nht.Pass("disabled management passes",
			fake.Role(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled),
			),
		),
		nht.Fail("enabled management fails",
			fake.Role(
				core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
			),
		),
		nht.Fail("invalid management fails",
			fake.Role(
				core.Annotation(v1.ResourceManagementKey, "invalid"),
			),
		),
	}

	nht.RunAll(t, nonhierarchical.ManagementAnnotationValidator, testCases)
}
