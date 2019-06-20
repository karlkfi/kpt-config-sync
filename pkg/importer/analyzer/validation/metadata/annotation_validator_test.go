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
	legalAnnotation    = "annotation"
	illegalAnnotation  = v1.ConfigManagementPrefix + "unsupported"
	illegalAnnotation2 = v1.ConfigManagementPrefix + "unsupported2"
)

func TestAnnotationValidator(t *testing.T) {
	test := asttest.Validator(NewAnnotationValidator,
		vet.IllegalAnnotationDefinitionErrorCode,

		asttest.Pass("no annotations",
			fake.Role(),
		),
		asttest.Pass("one legal annotation",
			fake.Role(
				object.Annotation(legalAnnotation, "")),
		),
		asttest.Fail("one illegal annotation",
			fake.Role(
				object.Annotation(illegalAnnotation, "")),
		),
		asttest.Fail("two illegal annotations",
			fake.Role(
				object.Annotation(illegalAnnotation, ""),
				object.Annotation(illegalAnnotation2, "")),
		),
		asttest.Fail("one legal and one illegal annotation",
			fake.Role(
				object.Annotation(legalAnnotation, ""),
				object.Annotation(illegalAnnotation, "")),
		),
		asttest.Pass("namespaceselector annotation",
			fake.Role(
				object.Annotation(v1.NamespaceSelectorAnnotationKey, "")),
		),
		asttest.Pass("clusterselector annotation",
			fake.Role(
				object.Annotation(v1.ClusterSelectorAnnotationKey, "")),
		),
		asttest.Pass("management annotation",
			fake.Role(
				object.Annotation(v1.ResourceManagementKey, "")),
		),
	)

	test.RunAll(t)
}

func TestNewManagedAnnotationValidator(t *testing.T) {
	test := asttest.Validator(NewManagedAnnotationValidator,
		vet.IllegalManagementAnnotationErrorCode,

		asttest.Pass("no management annotation",
			fake.Role(),
		),
		asttest.Pass("disabled management passes",
			fake.Role(
				object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled),
			),
		),
		asttest.Fail("enabled management fails",
			fake.Role(
				object.Annotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled),
			),
		),
		asttest.Fail("invalid management fails",
			fake.Role(
				object.Annotation(v1.ResourceManagementKey, "invalid"),
			),
		),
	)

	test.RunAll(t)
}
