package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDuplicateNameValidator(t *testing.T) {
	test := asttest.Validator(NewDuplicateNameValidator,
		vet.MetadataNameCollisionErrorCode,

		asttest.Pass("no objects passes"),
		asttest.Pass("one object passes",
			fake.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Namespace(), object.Path("namespaces/foo/ns.yaml")),
		),
		asttest.Fail("two colliding objects",
			fake.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Namespace(), object.Path("namespaces/foo/ns.yaml")),
		),
		asttest.Pass("two objects different group",
			fake.Build(fake.GVK(kinds.Role(), fake.Group("rbac")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(fake.GVK(kinds.Role(), fake.Group("google")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Namespace(), object.Path("namespaces/foo/ns.yaml")),
		),
		asttest.Fail("two colliding objects different version",
			fake.Build(fake.GVK(kinds.Role(), fake.Version("v1")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(fake.GVK(kinds.Role(), fake.Version("v2")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Namespace(), object.Path("namespaces/foo/ns.yaml")),
		),
		asttest.Pass("two objects different Kind",
			fake.Build(fake.GVK(kinds.Role(), fake.Kind("Role")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(fake.GVK(kinds.Role(), fake.Kind("RoleBinding")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Namespace(), object.Path("namespaces/foo/ns.yaml")),
		),
		asttest.Pass("two colliding objects in abstract namespace passes",
			fake.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
		),
		asttest.Pass("two objects different namespaces",
			fake.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			fake.Build(kinds.Namespace(), object.Path("namespaces/foo/ns.yaml")),
			fake.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/bar/role.yaml")),
			fake.Build(kinds.Namespace(), object.Path("namespaces/bar/ns.yaml")),
		),
		asttest.Fail("two colliding cluster/ objects",
			fake.Build(kinds.ClusterRole(),
				object.Name("clusterrole"), object.Path("cluster/clusterrole.yaml")),
			fake.Build(kinds.ClusterRole(),
				object.Name("clusterrole"), object.Path("cluster/clusterrole.yaml"))),
	)

	test.RunAll(t)
}
