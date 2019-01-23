package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet/vettesting"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type labelTestCase struct {
	name        string
	annotations []string
	error       []string
}

const (
	legalLabel    = "label"
	illegalLabel  = v1alpha1.NomosPrefix + "unsupported"
	illegalLabel2 = v1alpha1.NomosPrefix + "unsupported2"
)

var labelTestCases = []labelTestCase{
	{
		name: "no annotations",
	},
	{
		name:        "one legal annotation",
		annotations: []string{legalLabel},
	},
	{
		name:        "one illegal annotation",
		annotations: []string{illegalLabel},
		error:       []string{vet.IllegalLabelDefinitionErrorCode},
	},
	{
		name:        "two illegal annotations",
		annotations: []string{illegalLabel, illegalLabel2},
		error:       []string{vet.IllegalLabelDefinitionErrorCode},
	},
	{
		name:        "one legal and one illegal annotation",
		annotations: []string{legalLabel, illegalLabel},
		error:       []string{vet.IllegalLabelDefinitionErrorCode},
	},
	{
		name:        "namespaceselector annotation",
		annotations: []string{v1alpha1.NamespaceSelectorAnnotationKey},
		error:       []string{vet.IllegalLabelDefinitionErrorCode},
	},
	{
		name:        "clusterselector annotation",
		annotations: []string{v1alpha1.ClusterSelectorAnnotationKey},
		error:       []string{vet.IllegalLabelDefinitionErrorCode},
	},
}

func (tc labelTestCase) Run(t *testing.T) {
	annotations := make(map[string]string)
	for _, annotation := range tc.annotations {
		annotations[annotation] = ""
	}
	meta := resourceMeta{meta: &v1.ObjectMeta{Labels: annotations}}

	eb := multierror.Builder{}
	LabelValidatorFactory.New([]ResourceMeta{meta}).Validate(&eb)

	vettesting.ExpectErrors(tc.error, eb.Build(), t)
}

func TestLabelValidator(t *testing.T) {
	for _, tc := range labelTestCases {
		t.Run(tc.name, tc.Run)
	}
}
