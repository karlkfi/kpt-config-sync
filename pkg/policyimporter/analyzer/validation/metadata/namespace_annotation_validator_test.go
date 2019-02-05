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

func fakeAnnotatedRole(annotations ...string) ast.FileObject {
	object := asttesting.NewFakeObject(kinds.Role())
	object.Annotations = make(map[string]string)
	for _, annotation := range annotations {
		object.Annotations[annotation] = ""
	}
	return ast.FileObject{Object: object, Relative: nomospath.NewFakeRelative("namespaces/role.yaml")}
}

func fakeAnnotatedNamespace(annotations ...string) ast.FileObject {
	object := asttesting.NewFakeObject(kinds.Namespace())
	object.Annotations = make(map[string]string)
	for _, annotation := range annotations {
		object.Annotations[annotation] = ""
	}
	return ast.FileObject{Object: object, Relative: nomospath.NewFakeRelative("namespaces/namespace.yaml")}
}

func TestNamespaceAnnotationValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewNamespaceAnnotationValidator,
		ErrorCode: vet.IllegalNamespaceAnnotationErrorCode,
		TestCases: []visitortesting.ObjectValidatorTestCase{
			{
				Name:   "empty annotations",
				Object: fakeAnnotatedNamespace(),
			},
			{
				Name:   "legal annotation on Namespace",
				Object: fakeAnnotatedNamespace("legal"),
			},
			{
				Name:       "namespaceselector annotation on Namespace",
				Object:     fakeAnnotatedNamespace(v1alpha1.NamespaceSelectorAnnotationKey),
				ShouldFail: true,
			},
			{
				Name:   "namespaceselector annotation on Role",
				Object: fakeAnnotatedRole(v1alpha1.NamespaceSelectorAnnotationKey),
			},
			{
				Name:       "legal and namespaceselector annotations on Namespace",
				Object:     fakeAnnotatedNamespace("legal", v1alpha1.NamespaceSelectorAnnotationKey),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
