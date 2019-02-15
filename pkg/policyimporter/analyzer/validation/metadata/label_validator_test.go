package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/object"
)

const (
	legalLabel    = "label"
	illegalLabel  = v1alpha1.NomosPrefix + "unsupported"
	illegalLabel2 = v1alpha1.NomosPrefix + "unsupported2"
)

func TestLabelValidator(t *testing.T) {
	test := asttest.Validator(NewLabelValidator,
		vet.IllegalLabelDefinitionErrorCode,

		asttest.Pass("no labels",
			object.Build(kinds.Role()),
		),
		asttest.Pass("one legal label",
			object.Build(kinds.Role(),
				object.Label(legalLabel, "")),
		),
		asttest.Fail("one illegal label",
			object.Build(kinds.Role(),
				object.Label(illegalLabel, "")),
		),
		asttest.Fail("two illegal labels",
			object.Build(kinds.Role(),
				object.Label(illegalLabel, ""),
				object.Label(illegalLabel2, "")),
		),
		asttest.Fail("one legal and one illegal label",
			object.Build(kinds.Role(),
				object.Label(legalLabel, ""),
				object.Label(illegalLabel, "")),
		),
		asttest.Fail("namespaceselector label",
			object.Build(kinds.Role(),
				object.Label(v1alpha1.NamespaceSelectorAnnotationKey, "")),
		),
		asttest.Fail("clusterselector label",
			object.Build(kinds.Role(),
				object.Label(v1alpha1.ClusterSelectorAnnotationKey, "")),
		),
	)

	test.RunAll(t)
}
