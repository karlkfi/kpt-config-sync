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
	legalAnnotation    = "annotation"
	illegalAnnotation  = v1alpha1.NomosPrefix + "unsupported"
	illegalAnnotation2 = v1alpha1.NomosPrefix + "unsupported2"
)

func fakeAnnotatedObject(annotations ...string) ast.FileObject {
	object := asttesting.NewFakeObject(kinds.Role())
	object.Annotations = make(map[string]string)
	for _, annotation := range annotations {
		object.Annotations[annotation] = ""
	}
	return ast.FileObject{Object: object, Relative: nomospath.NewFakeRelative("namespaces/role.yaml")}
}

func TestAnnotationValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewAnnotationValidator,
		ErrorCode: vet.IllegalAnnotationDefinitionErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:   "no annotations",
				Object: fakeAnnotatedObject(),
			},
			{
				Name:   "one legal annotation",
				Object: fakeAnnotatedObject(legalAnnotation),
			},
			{
				Name:       "one illegal annotation",
				Object:     fakeAnnotatedObject(illegalAnnotation),
				ShouldFail: true,
			},
			{
				Name:       "two illegal annotations",
				Object:     fakeAnnotatedObject(illegalAnnotation, illegalAnnotation2),
				ShouldFail: true,
			},
			{
				Name:       "one legal and one illegal annotation",
				Object:     fakeAnnotatedObject(legalAnnotation, illegalAnnotation),
				ShouldFail: true,
			},
			{
				Name:   "namespaceselector annotation",
				Object: fakeAnnotatedObject(v1alpha1.NamespaceSelectorAnnotationKey),
			},
			{
				Name:   "clusterselector annotation",
				Object: fakeAnnotatedObject(v1alpha1.ClusterSelectorAnnotationKey),
			},
		},
	}

	test.RunAll(t)
}
