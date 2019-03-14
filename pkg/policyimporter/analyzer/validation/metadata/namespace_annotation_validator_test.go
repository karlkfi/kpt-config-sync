package metadata

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
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
	return ast.FileObject{Object: object, Path: nomospath.FromSlash("namespaces/role.yaml")}
}

func fakeAnnotatedNamespace(annotations ...string) ast.FileObject {
	object := asttesting.NewFakeObject(kinds.Namespace())
	object.Annotations = make(map[string]string)
	for _, annotation := range annotations {
		object.Annotations[annotation] = ""
	}
	return ast.FileObject{Object: object, Path: nomospath.FromSlash("namespaces/namespace.yaml")}
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
				Object:     fakeAnnotatedNamespace(v1.NamespaceSelectorAnnotationKey),
				ShouldFail: true,
			},
			{
				Name:   "namespaceselector annotation on Role",
				Object: fakeAnnotatedRole(v1.NamespaceSelectorAnnotationKey),
			},
			{
				Name:       "legal and namespaceselector annotations on Namespace",
				Object:     fakeAnnotatedNamespace("legal", v1.NamespaceSelectorAnnotationKey),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
