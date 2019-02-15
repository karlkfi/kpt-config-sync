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
	legalAnnotation    = "annotation"
	illegalAnnotation  = v1alpha1.NomosPrefix + "unsupported"
	illegalAnnotation2 = v1alpha1.NomosPrefix + "unsupported2"
)

func TestAnnotationValidator(t *testing.T) {
	test := asttest.Validator(NewAnnotationValidator,
		vet.IllegalAnnotationDefinitionErrorCode,

		asttest.Pass("no annotations",
			object.Build(kinds.Role()),
		),
		asttest.Pass("one legal annotation",
			object.Build(kinds.Role(),
				object.Annotation(legalAnnotation, "")),
		),
		asttest.Fail("one illegal annotation",
			object.Build(kinds.Role(),
				object.Annotation(illegalAnnotation, "")),
		),
		asttest.Fail("two illegal annotations",
			object.Build(kinds.Role(),
				object.Annotation(illegalAnnotation, ""),
				object.Annotation(illegalAnnotation2, "")),
		),
		asttest.Fail("one legal and one illegal annotation",
			object.Build(kinds.Role(),
				object.Annotation(legalAnnotation, ""),
				object.Annotation(illegalAnnotation, "")),
		),
		asttest.Pass("namespaceselector annotation",
			object.Build(kinds.Role(),
				object.Annotation(v1alpha1.NamespaceSelectorAnnotationKey, "")),
		),
		asttest.Pass("clusterselector annotation",
			object.Build(kinds.Role(),
				object.Annotation(v1alpha1.ClusterSelectorAnnotationKey, "")),
		),
	)

	test.RunAll(t)
}
