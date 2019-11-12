package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDuplicateNameValidator(t *testing.T) {
	test := asttest.Validator(NewDuplicateNameValidator,
		NameCollisionErrorCode,

		asttest.Pass("no objects passes"),
		asttest.Pass("one object passes",
			fake.RoleAtPath("namespaces/foo/role.yaml",
				core.Name("role")),
			fake.Namespace("namespaces/foo"),
		),
		asttest.Fail("two colliding objects",
			fake.RoleAtPath("namespaces/foo/role.yaml",
				core.Name("role")),
			fake.RoleAtPath("namespaces/foo/role.yaml",
				core.Name("role")),
			fake.Namespace("namespaces/foo"),
		),
		asttest.Pass("two objects different group",
			fake.UnstructuredAtPath(fake.GVK(kinds.Role(), fake.Group("rbac")),
				"namespaces/foo/role.yaml", core.Name("role")),
			fake.UnstructuredAtPath(fake.GVK(kinds.Role(), fake.Group("google")),
				"namespaces/foo/role.yaml", core.Name("role")),
			fake.Namespace("namespaces/foo"),
		),
		asttest.Fail("two colliding objects different version",
			fake.UnstructuredAtPath(fake.GVK(kinds.Role(), fake.Version("v1")),
				"namespaces/foo/role.yaml", core.Name("role")),
			fake.UnstructuredAtPath(fake.GVK(kinds.Role(), fake.Version("v2")),
				"namespaces/foo/role.yaml", core.Name("role")),
			fake.Namespace("namespaces/foo"),
		),
		asttest.Pass("two objects different Kind",
			fake.UnstructuredAtPath(fake.GVK(kinds.Role(), fake.Kind("Role")),
				"namespaces/foo/role.yaml", core.Name("role")),
			fake.UnstructuredAtPath(fake.GVK(kinds.Role(), fake.Kind("RoleBinding")),
				"namespaces/foo/role.yaml", core.Name("role")),
			fake.Namespace("namespaces/foo"),
		),
		asttest.Pass("two colliding objects in abstract namespace passes",
			fake.RoleAtPath("namespaces/foo/role.yaml",
				core.Name("role")),
			fake.RoleAtPath("namespaces/foo/role.yaml",
				core.Name("role")),
		),
		asttest.Pass("two objects different namespaces",
			fake.RoleAtPath("namespaces/foo/role.yaml",
				core.Name("role")),
			fake.Namespace("namespaces/foo"),
			fake.RoleAtPath("namespaces/bar/role.yaml",
				core.Name("role")),
			fake.Namespace("namespaces/bar"),
		),
		asttest.Fail("two colliding cluster/ objects",
			fake.ClusterRoleAtPath(
				"cluster/clusterrole.yaml", core.Name("clusterrole")),
			fake.ClusterRoleAtPath("cluster/clusterrole.yaml",
				core.Name("clusterrole"))),
	)

	test.RunAll(t)
}
