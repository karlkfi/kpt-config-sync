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
	"fmt"
	"testing"
	"time"

	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kpt.dev/configsync/e2e/nomostest"
	"kpt.dev/configsync/e2e/nomostest/metrics"
	"kpt.dev/configsync/e2e/nomostest/ntopts"
	nomostesting "kpt.dev/configsync/e2e/nomostest/testing"
	v1 "kpt.dev/configsync/pkg/api/configmanagement/v1"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/api/configsync/v1beta1"
	"kpt.dev/configsync/pkg/core"
	"kpt.dev/configsync/pkg/importer/analyzer/validation/system"
	"kpt.dev/configsync/pkg/kinds"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/reconcilermanager/controllers"
	"kpt.dev/configsync/pkg/testing/fake"
	"kpt.dev/configsync/pkg/validate/raw/validate"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteRootSyncAndRootSyncV1Alpha1(t *testing.T) {
	nt := nomostest.New(t, nomostesting.ACMController)

	var rs v1beta1.RootSync
	err := nt.Validate(configsync.RootSyncName, v1.NSConfigManagementSystem, &rs)
	if err != nil {
		nt.T.Fatal(err)
	}

	// Delete RootSync custom resource from the cluster.
	err = nt.Delete(&rs)
	if err != nil {
		nt.T.Fatalf("deleting RootSync: %v", err)
	}

	// Verify RootSync no longer present.
	if err := nomostest.WatchForNotFound(nt, kinds.RootSyncV1Beta1(), rs.GetName(), rs.GetNamespace()); err != nil {
		nt.T.Fatal(err)
	}

	// Verify Root Reconciler deployment no longer present.
	_, err = nomostest.Retry(40*time.Second, func() error {
		var errs error
		errs = multierr.Append(errs, nt.ValidateNotFound(nomostest.DefaultRootReconcilerName, v1.NSConfigManagementSystem, fake.DeploymentObject()))
		// validate Root Reconciler configmaps are no longer present.
		errs = multierr.Append(errs, nt.ValidateNotFound("root-reconciler-git-sync", v1.NSConfigManagementSystem, fake.ConfigMapObject()))
		errs = multierr.Append(errs, nt.ValidateNotFound("root-reconciler-reconciler", v1.NSConfigManagementSystem, fake.ConfigMapObject()))
		errs = multierr.Append(errs, nt.ValidateNotFound("root-reconciler-hydration-controller", v1.NSConfigManagementSystem, fake.ConfigMapObject()))
		errs = multierr.Append(errs, nt.ValidateNotFound("root-reconciler-source-format", v1.NSConfigManagementSystem, fake.ConfigMapObject()))
		// validate Root Reconciler ServiceAccount is no longer present.
		saName := core.RootReconcilerName(rs.Name)
		errs = multierr.Append(errs, nt.ValidateNotFound(saName, v1.NSConfigManagementSystem, fake.ServiceAccountObject(saName)))
		// validate Root Reconciler ClusterRoleBinding is no longer present.
		errs = multierr.Append(errs, nt.ValidateNotFound(controllers.RootSyncPermissionsName(), v1.NSConfigManagementSystem, fake.ClusterRoleBindingObject()))
		return errs
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Test RootSync v1alpha1 version")
	rsv1alpha1 := nomostest.RootSyncObjectV1Alpha1FromRootRepo(nt, configsync.RootSyncName)
	if err := nt.Create(rsv1alpha1); err != nil {
		nt.T.Fatal(err)
	}
	nt.WaitForRepoSyncs()
}

func TestUpdateRootSyncGitDirectory(t *testing.T) {
	nt := nomostest.New(t, nomostesting.SyncSource)

	// Validate RootSync is present.
	var rs v1beta1.RootSync
	err := nt.Validate(configsync.RootSyncName, v1.NSConfigManagementSystem, &rs)
	if err != nil {
		nt.T.Fatal(err)
	}

	// Add audit namespace in policy directory acme.
	acmeNS := "audit"
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("%s/namespaces/%s/ns.yaml", rs.Spec.Git.Dir, acmeNS),
		fake.NamespaceObject(acmeNS))

	// Add namespace in policy directory 'foo'.
	fooDir := "foo"
	fooNS := "shipping"
	sourcePath := fmt.Sprintf("%s/namespaces/%s/ns.yaml", fooDir, fooNS)
	nt.RootRepos[configsync.RootSyncName].Add(sourcePath, fake.NamespaceObject(fooNS))

	// Add repo resource in policy directory 'foo'.
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("%s/system/repo.yaml", fooDir),
		fake.RepoObject())

	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add namespace to acme directory")
	nt.WaitForRepoSyncs()

	// Validate namespace 'audit' created.
	err = nt.Validate(acmeNS, "", fake.NamespaceObject(acmeNS))
	if err != nil {
		nt.T.Error(err)
	}

	// Validate namespace 'shipping' not present.
	err = nt.ValidateNotFound(fooNS, "", fake.NamespaceObject(fooNS))
	if err != nil {
		nt.T.Errorf("%s present after deletion: %v", fooNS, err)
	}

	// Validate multi-repo metrics.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName,
			nt.DefaultRootSyncObjectCount()+1, // 1 for the test Namespace
			metrics.ResourceCreated("Namespace"))
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}

	// Update RootSync.
	nomostest.SetPolicyDir(nt, configsync.RootSyncName, fooDir)
	syncDirectoryMap := map[types.NamespacedName]string{nomostest.RootSyncNN(configsync.RootSyncName): fooDir}
	nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(syncDirectoryMap))

	// Validate namespace 'shipping' created with the correct sourcePath annotation.
	if err := nt.Validate(fooNS, "", fake.NamespaceObject(fooNS),
		nomostest.HasAnnotation(metadata.SourcePathAnnotationKey, sourcePath)); err != nil {
		nt.T.Error(err)
	}

	// Validate namespace 'audit' no longer present.
	// Namespace should be marked as deleted, but may not be NotFound yet,
	// because its finalizer will block until all objects in that namespace are
	// deleted.
	err = nomostest.WatchForNotFound(nt, kinds.Namespace(), acmeNS, "")
	if err != nil {
		nt.T.Error(err)
	}

	// Validate multi-repo metrics.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName,
			1, // 1 for the test Namespace
			metrics.GVKMetric{
				GVK:   "Namespace",
				APIOp: "delete",
				ApplyOps: []metrics.Operation{
					{Name: "delete", Count: 1},
				},
				Watches: "1",
			})
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}
}

func TestUpdateRootSyncGitBranch(t *testing.T) {
	nt := nomostest.New(t, nomostesting.SyncSource)

	// Add audit namespace.
	auditNS := "audit"
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", auditNS),
		fake.NamespaceObject(auditNS))
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add namespace to acme directory")
	nt.WaitForRepoSyncs()

	// Validate namespace 'acme' created.
	err := nt.Validate(auditNS, "", fake.NamespaceObject(auditNS))
	if err != nil {
		nt.T.Error(err)
	}

	testBranch := "test-branch"
	testNS := "audit-test"

	// Add a 'test-branch' branch with 'audit-test' namespace.
	nt.RootRepos[configsync.RootSyncName].CreateBranch(testBranch)
	nt.RootRepos[configsync.RootSyncName].CheckoutBranch(testBranch)
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", testNS),
		fake.NamespaceObject(testNS))
	nt.RootRepos[configsync.RootSyncName].CommitAndPushBranch("add audit-test to acme directory", testBranch)

	// Validate namespace 'audit-test' not present to vaidate rootsync is not syncing
	// from 'test-branch' yet.
	err = nt.ValidateNotFound(testNS, "", fake.NamespaceObject(testNS))
	if err != nil {
		nt.T.Errorf("%s present: %v", testNS, err)
	}

	// Set branch to "test-branch"
	nomostest.SetGitBranch(nt, configsync.RootSyncName, testBranch)

	nt.WaitForRepoSyncs()

	// Validate namespace 'audit-test' created after updating rootsync.
	err = nt.Validate(testNS, "", fake.NamespaceObject(testNS))
	if err != nil {
		nt.T.Error(err)
	}

	// Set branch to "main"
	nomostest.SetGitBranch(nt, configsync.RootSyncName, nomostest.MainBranch)

	// Checkout back to 'main' branch to get the correct HEAD commit sha1.
	nt.RootRepos[configsync.RootSyncName].CheckoutBranch(nomostest.MainBranch)
	nt.WaitForRepoSyncs()

	// Validate namespace 'acme' present.
	err = nt.Validate(auditNS, "", fake.NamespaceObject(auditNS))
	if err != nil {
		nt.T.Error(err)
	}

	// Validate namespace 'audit-test' not present to vaidate rootsync is not
	// syncing from 'test-branch' anymore.
	// Namespace should be marked as deleted, but may not be NotFound yet,
	// because its finalizer will block until all objects in that namespace are
	// deleted.
	err = nomostest.WatchForNotFound(nt, kinds.Namespace(), testNS, "")
	if err != nil {
		nt.T.Fatalf("RootSync update failed: %v", err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}
}

func TestForceRevert(t *testing.T) {
	nt := nomostest.New(t, nomostesting.SyncSource)

	nt.RootRepos[configsync.RootSyncName].Remove("acme/system/repo.yaml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Cause source error")

	nt.WaitForRootSyncSourceError(configsync.RootSyncName, system.MissingRepoErrorCode, "")

	err := nt.ValidateMetrics(nomostest.SyncMetricsToReconcilerSourceError(nt, nomostest.DefaultRootReconcilerName), func() error {
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(nomostest.DefaultRootReconcilerName, 1, 0)
	})
	if err != nil {
		nt.T.Error(err)
	}

	nt.RootRepos[configsync.RootSyncName].Git("reset", "--hard", "HEAD^")
	nt.RootRepos[configsync.RootSyncName].Git("push", "-f", "origin", "main")

	nt.WaitForRepoSyncs()

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateReconcilerErrors(nomostest.DefaultRootReconcilerName, 0, 0)
	})
	if err != nil {
		nt.T.Error(err)
	}
}

func TestRootSyncReconcilingStatus(t *testing.T) {
	nt := nomostest.New(t, nomostesting.ACMController)

	// Validate status condition "Reconciling" is set to "False" after the Reconciler
	// Deployment is successfully created.
	// Log error if the Reconciling condition does not progress to False before the timeout
	// expires.
	err := nomostest.WatchObject(nt, kinds.RootSyncV1Beta1(), configsync.RootSyncName, v1.NSConfigManagementSystem,
		[]nomostest.Predicate{
			hasRootSyncReconcilingStatus(metav1.ConditionFalse),
			hasRootSyncStalledStatus(metav1.ConditionFalse),
		},
		nomostest.WatchTimeout(15*time.Second))
	if err != nil {
		nt.T.Errorf("RootSync did not finish reconciling: %v", err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}
}

func TestManageSelfRootSync(t *testing.T) {
	nt := nomostest.New(t, nomostesting.ACMController, ntopts.Unstructured)
	rs := &v1beta1.RootSync{}
	if err := nt.Get(configsync.RootSyncName, configsync.ControllerNamespace, rs); err != nil {
		nt.T.Fatal(err)
	}
	sanitizedRs := fake.RootSyncObjectV1Beta1(rs.Name)
	sanitizedRs.Spec = rs.Spec
	nt.RootRepos[configsync.RootSyncName].Add("acme/root-sync.yaml", sanitizedRs)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add the root-sync object that configures the reconciler")
	nt.WaitForRootSyncSourceError(configsync.RootSyncName, validate.SelfReconcileErrorCode, "RootSync config-management-system/root-sync must not manage itself in its repo")
}

func hasRootSyncReconcilingStatus(r metav1.ConditionStatus) nomostest.Predicate {
	return func(o client.Object) error {
		rs := o.(*v1beta1.RootSync)
		conditions := rs.Status.Conditions
		for _, condition := range conditions {
			if condition.Type == "Reconciling" && condition.Status != r {
				return fmt.Errorf("object %q have %q condition status %q; wanted %q", o.GetName(), condition.Type, string(condition.Status), r)
			}
		}
		return nil
	}
}

func hasRootSyncStalledStatus(r metav1.ConditionStatus) nomostest.Predicate {
	return func(o client.Object) error {
		rs := o.(*v1beta1.RootSync)
		conditions := rs.Status.Conditions
		for _, condition := range conditions {
			if condition.Type == "Stalled" && condition.Status != r {
				return fmt.Errorf("object %q have %q condition status %q; wanted %q", o.GetName(), condition.Type, string(condition.Status), r)
			}
		}
		return nil
	}
}
