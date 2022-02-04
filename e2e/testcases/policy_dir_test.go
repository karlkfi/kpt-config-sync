package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/status"
)

func TestMissingRepoErrorWithHierarchicalFormat(t *testing.T) {
	nt := nomostest.New(t)

	nomostest.SetPolicyDir(nt, configsync.RootSyncName, "")

	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(configsync.RootSyncName, system.MissingRepoErrorCode, "")
	} else {
		nt.WaitForRepoImportErrorCode(system.MissingRepoErrorCode)
	}
}

func TestPolicyDirUnset(t *testing.T) {
	nt := nomostest.New(t)
	// There are 6 cluster-scoped objects under `../../examples/acme/cluster`.
	//
	// Copying the whole `../../examples/acme/cluster` dir would cause the Config Sync mono-repo mode CI job to fail,
	// which runs on a shared cluster and calls resetSyncedRepos at the end of every e2e test.
	//
	// The reason for the failure is that if there are more than 1 cluster-scoped objects in a Git repo,
	// Config Sync mono-repo mode does not allow removing all these cluster-scoped objects in a single commit,
	// and generates a KNV 2006 error (as shown in http://b/210525686#comment3 and http://b/210525686#comment5).
	//
	// Therefore, we only copy `../../examples/acme/cluster/admin-clusterrole.yaml` here.
	nt.RootRepos[configsync.RootSyncName].Copy("../../examples/acme/cluster/admin-clusterrole.yaml", "./cluster")
	nt.RootRepos[configsync.RootSyncName].Copy("../../examples/acme/namespaces", ".")
	nt.RootRepos[configsync.RootSyncName].Copy("../../examples/acme/system", ".")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Initialize the root directory")
	nt.WaitForRepoSyncs()

	nomostest.SetPolicyDir(nt, configsync.RootSyncName, "")
	nt.WaitForRepoSyncs(nomostest.WithSyncDirectory("."))
}

func TestInvalidPolicyDir(t *testing.T) {
	nt := nomostest.New(t)

	nt.T.Log("Break the policydir in the repo")
	nomostest.SetPolicyDir(nt, configsync.RootSyncName, "some-nonexistent-policydir")

	nt.T.Log("Expect an error to be present in status.source.errors")
	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(configsync.RootSyncName, status.PathErrorCode, "")
	} else {
		nt.WaitForRepoSourceError(status.SourceErrorCode)
	}

	nt.T.Log("Fix the policydir in the repo")
	nomostest.SetPolicyDir(nt, configsync.RootSyncName, "acme")
	nt.T.Log("Expect repo to recover from the error in source message")
	nt.WaitForRepoSyncs()
}
