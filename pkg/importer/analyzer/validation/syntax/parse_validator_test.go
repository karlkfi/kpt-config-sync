package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestParseValidator(t *testing.T) {
	test := asttest.Validator(NewParseValidator,
		vet.ObjectParseErrorCode,
		asttest.Pass("cluster",
			fake.Build(kinds.Cluster()),
		),
		asttest.Fail("invalid cluster",
			fake.Unstructured(kinds.Cluster(), object.Path("cluster/c.yaml")),
		),
		asttest.Pass("hierarchyconfig",
			fake.Build(kinds.HierarchyConfig()),
		),
		asttest.Fail("invalid hierarchyconfig",
			fake.Unstructured(kinds.HierarchyConfig(), object.Path("system/hc.yaml")),
		),
		asttest.Pass("namespaceselector",
			fake.Build(kinds.NamespaceSelector()),
		),
		asttest.Fail("invalid namespaceselector",
			fake.Unstructured(kinds.NamespaceSelector(), object.Path("namespaces/ns.yaml")),
		),
		asttest.Pass("repo",
			fake.Build(kinds.Repo()),
		),
		asttest.Fail("invalid repo",
			fake.Unstructured(kinds.Repo(), object.Path("system/repo.yaml")),
		),
	)

	test.RunAll(t)
}
