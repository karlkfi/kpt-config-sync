package e2e

import (
	"fmt"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestMissingRepoErrorWithHierarchicalFormat(t *testing.T) {
	nt := nomostest.New(t)

	setPolicyDir(nt, "")

	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(system.MissingRepoErrorCode)
	} else {
		nt.WaitForRepoImportErrorCode(system.MissingRepoErrorCode)
	}
}

func TestPolicyDirUnset(t *testing.T) {
	nt := nomostest.New(t)
	nt.Root.Copy("../../examples/acme/cluster", ".")
	nt.Root.Copy("../../examples/acme/namespaces", ".")
	nt.Root.Copy("../../examples/acme/system", ".")
	nt.Root.CommitAndPush("Initialize the root directory")
	nt.WaitForRepoSyncs()

	setPolicyDir(nt, "")
	nt.WaitForRepoSyncs()
}

func TestInvalidPolicyDir(t *testing.T) {
	nt := nomostest.New(t)

	nt.T.Log("Break the policydir in the repo")
	setPolicyDir(nt, "some-nonexistent-policydir")

	nt.T.Log("Expect an error to be present in status.source.errors")
	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(status.PathErrorCode)
	} else {
		nt.WaitForRepoSourceError(status.SourceErrorCode)
	}

	nt.T.Log("Fix the policydir in the repo")
	setPolicyDir(nt, "acme")
	nt.T.Log("Expect repo to recover from the error in source message")
	nt.WaitForRepoSyncs()
}

func setPolicyDir(nt *nomostest.NT, policyDir string) {
	nt.T.Logf("Set policyDir to %q", policyDir)
	if nt.MultiRepo {
		rs := fake.RootSyncObject()
		nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "%s"}}}`, policyDir))
	} else {
		nomostest.ResetMonoRepoSpec(nt, filesystem.SourceFormatHierarchy, policyDir)
	}
}
