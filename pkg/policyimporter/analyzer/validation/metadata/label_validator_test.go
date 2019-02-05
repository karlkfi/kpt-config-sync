package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	visitortesting "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
)

const (
	legalLabel    = "annotation"
	illegalLabel  = v1alpha1.NomosPrefix + "unsupported"
	illegalLabel2 = v1alpha1.NomosPrefix + "unsupported2"
)

func fakeLabeledObject(labels ...string) ast.FileObject {
	result := asttesting.NewFakeObject(kinds.Role())
	result.Labels = make(map[string]string)
	for _, label := range labels {
		result.Labels[label] = ""
	}
	return ast.FileObject{Object: result, Relative: nomospath.NewFakeRelative("namespaces/role.yaml")}
}

func TestLabelValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewLabelValidator,
		ErrorCode: vet.IllegalLabelDefinitionErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:   "no annotations",
				Object: fakeLabeledObject(),
			},
			{
				Name:   "one legal annotation",
				Object: fakeLabeledObject(legalLabel),
			},
			{
				Name:       "one illegal annotation",
				Object:     fakeLabeledObject(illegalLabel),
				ShouldFail: true,
			},
			{
				Name:       "two illegal annotations",
				Object:     fakeLabeledObject(illegalLabel, illegalLabel2),
				ShouldFail: true,
			},
			{
				Name:       "one legal and one illegal annotation",
				Object:     fakeLabeledObject(legalLabel, illegalLabel),
				ShouldFail: true,
			},
			{
				Name:       "namespaceselector annotation",
				Object:     fakeLabeledObject(v1alpha1.NamespaceSelectorAnnotationKey),
				ShouldFail: true,
			},
			{
				Name:       "clusterselector annotation",
				Object:     fakeLabeledObject(v1alpha1.ClusterSelectorAnnotationKey),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
