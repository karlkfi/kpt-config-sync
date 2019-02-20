package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/object"
)

func TestDuplicateNameValidator(t *testing.T) {
	test := asttest.Validator(NewDuplicateNameValidator,
		vet.MetadataNameCollisionErrorCode,

		asttest.Pass("no objects passes"),
		asttest.Pass("one object passes",
			object.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
		),
		asttest.Fail("two colliding objects",
			object.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			object.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
		),
		asttest.Pass("two objects different group",
			object.Build(object.GVK(kinds.Role(), object.Group("rbac")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			object.Build(object.GVK(kinds.Role(), object.Group("google")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
		),
		asttest.Fail("two colliding objects different version",
			object.Build(object.GVK(kinds.Role(), object.Version("v1")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			object.Build(object.GVK(kinds.Role(), object.Version("v2")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
		),
		asttest.Pass("two objects different Kind",
			object.Build(object.GVK(kinds.Role(), object.Kind("Role")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			object.Build(object.GVK(kinds.Role(), object.Kind("RoleBinding")),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
		),
		asttest.Fail("two colliding objects in abstract namespace",
			object.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			object.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
		),
		asttest.Pass("two objects different namespaces",
			object.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/foo/role.yaml")),
			object.Build(kinds.Role(),
				object.Name("role"), object.Path("namespaces/bar/role.yaml")),
		),
		asttest.Fail("two colliding cluster/ objects",
			object.Build(kinds.ClusterRole(),
				object.Name("clusterrole"), object.Path("cluster/clusterrole.yaml")),
			object.Build(kinds.ClusterRole(),
				object.Name("clusterrole"), object.Path("cluster/clusterrole.yaml"))),
	)

	test.RunAll(t)
}
