package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestNamespaceRepo_Centralized(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	rs := fake.RepoSyncObject(core.Namespace("foo"))
	rs.Spec.Repo = "TODO(b/168915318)"
	rs.Spec.Auth = "none"

	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/repo-sync.yaml", rs)
	nt.Root.CommitAndPush("Adding foo namespace and RepoSync")
	nt.WaitForRepoSync()

	err := nt.Validate(rs.Name, "foo", &v1alpha1.RepoSync{})
	if err != nil {
		t.Error(err)
	}

	// TODO(b/168915318): Validate that the objects in the namespace repo get synced.
}
