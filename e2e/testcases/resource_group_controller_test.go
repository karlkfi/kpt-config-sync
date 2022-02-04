package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/resourcegroup"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestResourceGroupController(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.InstallResourceGroupController)

	ns := "rg-test"
	nt.RootRepos[configsync.RootSyncName].Add("acme/namespaces/rg-test/ns.yaml",
		fake.NamespaceObject(ns))

	cmName := "e2e-test-configmap"
	cmPath := "acme/namespaces/rg-test/configmap.yaml"
	cm := fake.ConfigMapObject(core.Name(cmName), core.Namespace(ns))
	nt.RootRepos[configsync.RootSyncName].Add(cmPath, cm)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Adding a ConfigMap to repo")
	nt.WaitForRepoSyncs()

	// Checking that the ResourceGroup controller captures the status of the
	// managed resources.
	id := applier.InventoryID(configsync.ControllerNamespace)
	_, err := nomostest.Retry(60*time.Second, func() error {
		rg := resourcegroup.Unstructured(configsync.RootSyncName, configsync.ControllerNamespace, id)
		err := nt.Validate(configsync.RootSyncName, configsync.ControllerNamespace, rg,
			nomostest.AllResourcesAreCurrent())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
