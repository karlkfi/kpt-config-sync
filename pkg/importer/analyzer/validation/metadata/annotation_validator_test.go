package metadata

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

const (
	legalAnnotation    = "annotation"
	illegalAnnotation  = v1.ConfigManagementPrefix + "unsupported"
	illegalAnnotation2 = v1.ConfigManagementPrefix + "unsupported2"
)

func TestAnnotationValidator(t *testing.T) {
	asttest.Validator(t, NewAnnotationValidator,
		IllegalAnnotationDefinitionErrorCode,

		asttest.Pass("no annotations",
			fake.Role(),
		),
		asttest.Pass("one legal annotation",
			fake.Role(
				core.Annotation(legalAnnotation, "")),
		),
		asttest.Fail("one illegal annotation",
			fake.Role(
				core.Annotation(illegalAnnotation, "")),
		),
		asttest.Fail("two illegal annotations",
			fake.Role(
				core.Annotation(illegalAnnotation, ""),
				core.Annotation(illegalAnnotation2, "")),
		),
		asttest.Fail("one legal and one illegal annotation",
			fake.Role(
				core.Annotation(legalAnnotation, ""),
				core.Annotation(illegalAnnotation, "")),
		),
		asttest.Pass("namespaceselector annotation",
			fake.Role(
				core.Annotation(v1.NamespaceSelectorAnnotationKey, "")),
		),
		asttest.Pass("clusterselector annotation",
			fake.Role(
				core.Annotation(v1.ClusterSelectorAnnotationKey, "")),
		),
		asttest.Pass("management annotation",
			fake.Role(
				core.Annotation(v1.ResourceManagementKey, "")),
		),
	)
}
