package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type annotationTestCase struct {
	name        string
	annotations []string
	error       []string
}

const (
	legalAnnotation    = "annotation"
	illegalAnnotation  = v1alpha1.NomosPrefix + "unsupported"
	illegalAnnotation2 = v1alpha1.NomosPrefix + "unsupported2"
)

var annotationTestCases = []annotationTestCase{
	{
		name: "no annotations",
	},
	{
		name:        "one legal annotation",
		annotations: []string{legalAnnotation},
	},
	{
		name:        "one illegal annotation",
		annotations: []string{illegalAnnotation},
		error:       []string{vet.IllegalAnnotationDefinitionErrorCode},
	},
	{
		name:        "two illegal annotations",
		annotations: []string{illegalAnnotation, illegalAnnotation2},
		error:       []string{vet.IllegalAnnotationDefinitionErrorCode},
	},
	{
		name:        "one legal and one illegal annotation",
		annotations: []string{legalAnnotation, illegalAnnotation},
		error:       []string{vet.IllegalAnnotationDefinitionErrorCode},
	},
	{
		name:        "namespaceselector annotation",
		annotations: []string{v1alpha1.NamespaceSelectorAnnotationKey},
	},
	{
		name:        "clusterselector annotation",
		annotations: []string{v1alpha1.ClusterSelectorAnnotationKey},
	},
}

func (tc annotationTestCase) Run(t *testing.T) {
	annotations := make(map[string]string)
	for _, annotation := range tc.annotations {
		annotations[annotation] = ""
	}
	meta := resourceMeta{meta: &v1.ObjectMeta{Annotations: annotations}}

	eb := multierror.Builder{}
	AnnotationValidatorFactory.New([]ResourceMeta{meta}).Validate(&eb)

	vettesting.ExpectErrors(tc.error, eb.Build(), t)
}

func TestAnnotationValidator(t *testing.T) {
	for _, tc := range annotationTestCases {
		t.Run(tc.name, tc.Run)
	}
}
