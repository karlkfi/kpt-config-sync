package metadata

import (
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	visitortesting "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
)

func fakeAnnotatedRole(annotations ...string) ast.FileObject {
	object := asttesting.NewFakeObject(kinds.Role())
	object.Annotations = make(map[string]string)
	for _, annotation := range annotations {
		object.Annotations[annotation] = ""
	}
	return ast.NewFileObject(object, cmpath.FromSlash("namespaces/role.yaml"))
}

func fakeAnnotatedNamespace(annotations ...string) ast.FileObject {
	object := asttesting.NewFakeObject(kinds.Namespace())
	object.Annotations = make(map[string]string)
	for _, annotation := range annotations {
		object.Annotations[annotation] = ""
	}
	return ast.NewFileObject(object, cmpath.FromSlash("namespaces/namespace.yaml"))
}

func TestNamespaceAnnotationValidator(t *testing.T) {
	test := visitortesting.ObjectValidatorTest{
		Validator: NewNamespaceAnnotationValidator,
		ErrorCode: IllegalNamespaceAnnotationErrorCode,
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
				Name:       "namespace-selector annotation on Namespace",
				Object:     fakeAnnotatedNamespace(v1.NamespaceSelectorAnnotationKey),
				ShouldFail: true,
			},
			{
				Name:   "namespace-selector annotation on Role",
				Object: fakeAnnotatedRole(v1.NamespaceSelectorAnnotationKey),
			},
			{
				Name:       "legal and namespace-selector annotations on Namespace",
				Object:     fakeAnnotatedNamespace("legal", v1.NamespaceSelectorAnnotationKey),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
