package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestMustRemoveCustomResourceWithDefinition(t *testing.T) {
	nt := nomostest.New(t)

	nt.Root.Add("acme/cluster/anvil-crd.yaml", anvilV1CRD())
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 10))
	nt.Root.CommitAndPush("Adding Anvil CRD and one Anvil CR")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	err := nt.Validate("heavy", "foo", anvilCR("v1", "", 0))
	if err != nil {
		t.Fatal(err)
	}

	// This should cause an error.
	nt.Root.Remove("acme/cluster/anvil-crd.yaml")
	nt.Root.CommitAndPush("Removing Anvil CRD but leaving Anvil CR")

	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(nonhierarchical.UnsupportedCRDRemovalErrorCode, "Custom Resources MUST be removed")
	} else {
		nt.WaitForRepoImportErrorCode(nonhierarchical.UnsupportedCRDRemovalErrorCode)
	}

	// This should fix the error.
	nt.Root.Remove("acme/namespaces/foo/anvil-v1.yaml")
	nt.Root.CommitAndPush("Removing the Anvil CR as well")
	nt.WaitForRepoSyncs()
}
