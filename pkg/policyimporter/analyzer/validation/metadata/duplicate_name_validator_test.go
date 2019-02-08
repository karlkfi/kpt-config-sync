package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	testing2 "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/api/rbac/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func namedRole(name string, path string) ast.FileObject {
	role := fake.Role(path)
	role.Object.(*v1alpha1.Role).SetName(name)
	return role
}

func namedRoleBinding(name string, path string) ast.FileObject {
	role := fake.RoleBinding(path)
	role.Object.(*v1alpha1.RoleBinding).SetName(name)
	return role
}

func namedRoleWithGroup(name string, group string, path string) ast.FileObject {
	o := asttesting.NewFakeObject(schema.GroupVersionKind{
		Group:   group,
		Version: kinds.Role().Version,
		Kind:    kinds.Role().Kind,
	})
	o.SetName(name)
	return ast.NewFileObject(o, nomospath.NewFakeRelative(path))
}

func namedRoleWithVersion(name string, version string, path string) ast.FileObject {
	o := asttesting.NewFakeObject(schema.GroupVersionKind{
		Group:   kinds.Role().Group,
		Version: version,
		Kind:    kinds.Role().Kind,
	})
	o.SetName(name)
	return ast.NewFileObject(o, nomospath.NewFakeRelative(path))
}

func TestDuplicateNameValidator(t *testing.T) {
	test := testing2.ObjectsValidatorTest{
		Validator: NewDuplicateNameValidator,
		ErrorCode: vet.MetadataNameCollisionErrorCode,
		TestCases: []testing2.ObjectsValidatorTestCase{
			{
				Name: "no objects passes",
			},
			{
				Name: "one object passes validation",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
					namedRole("a", "namespaces/role.yaml"),
				},
			},
			{
				Name: "two colliding objects",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
					namedRole("a", "namespaces/foo/role.yaml"),
					namedRole("a", "namespaces/foo/role.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "two colliding objects",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
					namedRole("a", "namespaces/foo/role.yaml"),
					namedRole("a", "namespaces/foo/role.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "two colliding objects different version",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
					namedRoleWithVersion("a", "v1", "namespaces/foo/role.yaml"),
					namedRoleWithVersion("a", "v2", "namespaces/foo/role.yaml"),
				},
				ShouldFail: true,
			},
			{
				Name: "two objects different group",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
					namedRoleWithGroup("a", "foo", "namespaces/foo/role.yaml"),
					namedRoleWithGroup("a", "bar", "namespaces/foo/role.yaml"),
				},
			},
			{
				Name: "two objects different Kind",
				Objects: []ast.FileObject{
					fake.Namespace("namespaces/foo/ns.yaml"),
					namedRole("a", "namespaces/foo/role.yaml"),
					namedRoleBinding("a", "namespaces/foo/rolebinding.yaml"),
				},
			},
			{
				Name: "two colliding objects in abstract namespace",
				Objects: []ast.FileObject{
					namedRole("a", "namespaces/foo/role.yaml"),
					namedRole("a", "namespaces/foo/role.yaml"),
				},
			},
			{
				Name: "two objects different namespaces",
				Objects: []ast.FileObject{
					// foo
					fake.Namespace("namespaces/foo/ns.yaml"),
					namedRole("a", "namespaces/foo/role.yaml"),
					// bar
					fake.Namespace("namespaces/bar/ns.yaml"),
					namedRole("a", "namespaces/bar/role.yaml"),
				},
			},
			{
				Name: "two colliding cluster/ objects",
				Objects: []ast.FileObject{
					namedRole("a", "cluster/role.yaml"),
					namedRole("a", "cluster/role.yaml"),
				},
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
