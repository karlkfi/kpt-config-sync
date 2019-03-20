package metadata

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
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
			fake.Build(kinds.Role()),
		),
		asttest.Pass("one legal label",
			fake.Build(kinds.Role(),
				object.Label(legalLabel, "")),
		),
		asttest.Fail("one illegal label",
			fake.Build(kinds.Role(),
				object.Label(illegalLabel, "")),
		),
		asttest.Fail("two illegal labels",
			fake.Build(kinds.Role(),
				object.Label(illegalLabel, ""),
				object.Label(illegalLabel2, "")),
		),
		asttest.Fail("one legal and one illegal label",
			fake.Build(kinds.Role(),
				object.Label(legalLabel, ""),
				object.Label(illegalLabel, "")),
		),
		asttest.Fail("namespaceselector label",
			fake.Build(kinds.Role(),
				object.Label(v1.NamespaceSelectorAnnotationKey, "")),
		),
		asttest.Fail("clusterselector label",
			fake.Build(kinds.Role(),
				object.Label(v1.ClusterSelectorAnnotationKey, "")),
		),
	)

	test.RunAll(t)
}
