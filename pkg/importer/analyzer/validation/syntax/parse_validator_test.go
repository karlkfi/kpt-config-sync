package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestParseValidator(t *testing.T) {
	test := asttest.Validator(NewParseValidator,
		vet.ObjectParseErrorCode,
		asttest.Pass("cluster",
			fake.Cluster(),
		),
		asttest.Fail("invalid cluster",
			fake.UnstructuredAtPath(kinds.Cluster(), "cluster/c.yaml"),
		),
		asttest.Pass("hierarchyconfig",
			fake.HierarchyConfig(),
		),
		asttest.Fail("invalid hierarchyconfig",
			fake.UnstructuredAtPath(kinds.HierarchyConfig(), "system/hc.yaml"),
		),
		asttest.Pass("namespaceselector",
			fake.NamespaceSelector(),
		),
		asttest.Fail("invalid namespaceselector",
			fake.UnstructuredAtPath(kinds.NamespaceSelector(), "namespaces/ns.yaml"),
		),
		asttest.Pass("repo",
			fake.Repo(),
		),
		asttest.Fail("invalid repo",
			fake.UnstructuredAtPath(kinds.Repo(), "system/repo.yaml"),
		),
	)

	test.RunAll(t)
}
