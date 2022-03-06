// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// This file includes testcases to enable status
// or disable status in the kpt applier.

func TestStatusEnabledAndDisabled(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured, ntopts.SkipMonoRepo)
	id := applier.InventoryID(configsync.RootSyncName, configsync.ControllerNamespace)

	rootSync := fake.RootSyncObjectV1Alpha1(configsync.RootSyncName)
	// Override the statusMode for root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"statusMode": "disabled"}}}`)
	nt.WaitForRepoSyncs()

	namespaceName := "status-test"
	nt.RootRepos[configsync.RootSyncName].Add("acme/ns.yaml", namespaceObject(namespaceName, nil))
	nt.RootRepos[configsync.RootSyncName].Add("acme/cm1.yaml", fake.ConfigMapObject(core.Name("cm1"), core.Namespace(namespaceName)))
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Add a namespace and a configmap")
	nt.WaitForRepoSyncs()

	_, err := nomostest.Retry(120*time.Second, func() error {
		rg := resourcegroup.Unstructured(configsync.RootSyncName, configsync.ControllerNamespace, id)
		err := nt.Validate(configsync.RootSyncName, configsync.ControllerNamespace, rg,
			nomostest.NoStatus())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the statusMode for root-reconciler to re-enable the status
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"statusMode": "enabled"}}}`)
	nt.WaitForRepoSyncs()

	_, err = nomostest.Retry(120*time.Second, func() error {
		rg := resourcegroup.Unstructured(configsync.RootSyncName, configsync.ControllerNamespace, id)
		err := nt.Validate(configsync.RootSyncName, configsync.ControllerNamespace, rg,
			nomostest.HasStatus())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
