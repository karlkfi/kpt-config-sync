package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestMissingRepoErrorWithHierarchicalFormat(t *testing.T) {
	nt := nomostest.New(t)

	setNoPolicyDir(nt)

	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError("1017")
	} else {
		nt.WaitForRepoImportErrorCode("1017")
	}
}

func TestPolicyDirUnset(t *testing.T) {
	nt := nomostest.New(t)
	nt.Root.Copy("../../examples/acme/cluster", ".")
	nt.Root.Copy("../../examples/acme/namespaces", ".")
	nt.Root.Copy("../../examples/acme/system", ".")
	nt.Root.CommitAndPush("Initialize the root directory")
	nt.WaitForRepoSyncs()

	setNoPolicyDir(nt)
	nt.WaitForRepoSyncs()
}

func setNoPolicyDir(nt *nomostest.NT) {
	if nt.MultiRepo {
		rs := fake.RootSyncObject()
		nt.MustMergePatch(rs, `{"spec": {"git": {"dir": ""}}}`)
	} else {
		nomostest.ResetMonoRepoSpec(nt, filesystem.SourceFormatHierarchy, "")
	}
}
