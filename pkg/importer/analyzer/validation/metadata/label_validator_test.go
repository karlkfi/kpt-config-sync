package metadata

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	legalLabel    = "label"
	illegalLabel  = v1.ConfigManagementPrefix + "unsupported"
	illegalLabel2 = v1.ConfigManagementPrefix + "unsupported2"
)

func TestLabelValidator(t *testing.T) {
	test := asttest.Validator(NewLabelValidator,
		vet.IllegalLabelDefinitionErrorCode,

		asttest.Pass("no labels",
			fake.Role(),
		),
		asttest.Pass("one legal label",
			fake.Role(
				object.Label(legalLabel, "")),
		),
		asttest.Fail("one illegal label",
			fake.Role(
				object.Label(illegalLabel, "")),
		),
		asttest.Fail("two illegal labels",
			fake.Role(
				object.Label(illegalLabel, ""),
				object.Label(illegalLabel2, "")),
		),
		asttest.Fail("one legal and one illegal label",
			fake.Role(
				object.Label(legalLabel, ""),
				object.Label(illegalLabel, "")),
		),
		asttest.Fail("namespaceselector label",
			fake.Role(
				object.Label(v1.NamespaceSelectorAnnotationKey, "")),
		),
		asttest.Fail("clusterselector label",
			fake.Role(
				object.Label(v1.ClusterSelectorAnnotationKey, "")),
		),
	)

	test.RunAll(t)
}
