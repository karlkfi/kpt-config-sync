// Package hnc adds additional HNC-understandable annotation and labels to namespaces managed by
// ACM. Please send code reviews to gke-kubernetes-hnc-core@.
package hnc

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	legalLabel            = "label"
	illegalSuffixedLabel  = "unsupported" + v1.HierarchyControllerDepthSuffix
	illegalSuffixedLabel2 = "unsupported2" + v1.HierarchyControllerDepthSuffix
)

func TestDepthLabelValidator(t *testing.T) {
	asttest.Validator(t, NewDepthLabelValidator,
		IllegalDepthLabelErrorCode,

		asttest.Pass("no labels",
			fake.Role(),
		),
		asttest.Pass("one legal label",
			fake.Role(
				core.Label(legalLabel, "")),
		),
		asttest.Fail("one illegal label",
			fake.Role(
				core.Label(illegalSuffixedLabel, "")),
		),
		asttest.Fail("two illegal labels",
			fake.Role(
				core.Label(illegalSuffixedLabel, ""),
				core.Label(illegalSuffixedLabel2, "")),
		),
		asttest.Fail("one legal and one illegal label",
			fake.Role(
				core.Label(legalLabel, ""),
				core.Label(illegalSuffixedLabel, "")),
		),
	)
}
