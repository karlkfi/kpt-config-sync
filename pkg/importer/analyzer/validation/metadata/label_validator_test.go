package metadata

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	legalLabel    = "label"
	illegalLabel  = v1.ConfigManagementPrefix + "unsupported"
	illegalLabel2 = v1alpha1.ConfigSyncPrefix + "unsupported2"
)

func TestLabelValidator(t *testing.T) {
	asttest.Validator(t, NewLabelValidator,
		IllegalLabelDefinitionErrorCode,

		asttest.Pass("no labels",
			fake.Role(),
		),
		asttest.Pass("one legal label",
			fake.Role(
				core.Label(legalLabel, "")),
		),
		asttest.Fail("one illegal label starts with `configmanagement.gke.io/`",
			fake.Role(
				core.Label(illegalLabel, "")),
		),
		asttest.Fail("one illegal label starts with `configsync.gke.io/`",
			fake.Role(
				core.Label(illegalLabel2, "")),
		),
		asttest.Fail("two illegal labels",
			fake.Role(
				core.Label(illegalLabel, ""),
				core.Label(illegalLabel2, "")),
		),
		asttest.Fail("one legal and one illegal label",
			fake.Role(
				core.Label(legalLabel, ""),
				core.Label(illegalLabel, "")),
		),
		asttest.Fail("namespaceselector label",
			fake.Role(
				core.Label(v1.NamespaceSelectorAnnotationKey, "")),
		),
		asttest.Fail("clusterselector label",
			fake.Role(
				core.Label(v1.LegacyClusterSelectorAnnotationKey, "")),
		),
		asttest.Fail("inline clusterselector label",
			fake.Role(
				core.Label(v1alpha1.ClusterNameSelectorAnnotationKey, "")),
		),
	)
}
