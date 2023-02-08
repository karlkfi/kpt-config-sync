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

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"kpt.dev/configsync/e2e"
	"kpt.dev/configsync/e2e/nomostest"
	"kpt.dev/configsync/e2e/nomostest/metrics"
	"kpt.dev/configsync/e2e/nomostest/ntopts"
	"kpt.dev/configsync/e2e/nomostest/policy"
	"kpt.dev/configsync/e2e/nomostest/taskgroup"
	nomostesting "kpt.dev/configsync/e2e/nomostest/testing"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/api/configsync/v1beta1"
	"kpt.dev/configsync/pkg/applier"
	"kpt.dev/configsync/pkg/core"
	"kpt.dev/configsync/pkg/declared"
	"kpt.dev/configsync/pkg/importer/filesystem"
	"kpt.dev/configsync/pkg/kinds"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/status"
	"kpt.dev/configsync/pkg/testing/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// testNs is the namespace of all RepoSync objects.
const testNs = "test-ns"

// TestMultiSyncs_Unstructured_MixedControl tests multiple syncs created in the mixed control mode.
// - root-sync is created using k8s api.
// - rr1 is created using k8s api. This is to validate multiple RootSyncs can be created in the delegated mode.
// - rr2 is a v1alpha1 version of RootSync declared in the root repo of root-sync. This is to validate RootSync can be managed in a root repo and validate the v1alpha1 version API.
// - rr3 is a v1alpha1 version of RootSync declared in rs-2. This is to validate RootSync can be managed in a different root repo and validate the v1alpha1 version API.
// - nr1 is created using k8s api. This is to validate RepoSyncs can be created in the delegated mode.
// - nr2 is a v1alpha1 version of RepoSync created using k8s api. This is to validate v1alpha1 version of RepoSync can be created in the delegated mode.
// - nr3 is declared in the root repo of root-sync. This is to validate RepoSync can be managed in a root repo.
// - nr4 is a v1alpha1 version of RepoSync declared in the namespace repo of nn2. This is to validate RepoSync can be managed in a namespace repo in the same namespace.
// - nr5 is declared in the root repo of rr1. This is to validate implicit namespace won't cause conflict between two root reconcilers (rr1 and root-sync).
// - nr6 is created using k8s api in a different namespace but with the same name "nr1".
func TestMultiSyncs_Unstructured_MixedControl(t *testing.T) {
	rr1 := "rr1"
	rr2 := "rr2"
	rr3 := "rr3"
	nr1 := "nr1"
	nn1 := nomostest.RepoSyncNN(testNs, nr1)
	nn2 := nomostest.RepoSyncNN(testNs, "nr2")
	nn3 := nomostest.RepoSyncNN(testNs, "nr3")
	nn4 := nomostest.RepoSyncNN(testNs, "nr4")
	nn5 := nomostest.RepoSyncNN(testNs, "nr5")
	testNs2 := "ns-2"
	nn6 := nomostest.RepoSyncNN(testNs2, nr1)

	nt := nomostest.New(t, nomostesting.MultiRepos, ntopts.Unstructured,
		ntopts.WithDelegatedControl, ntopts.RootRepo(rr1),
		// NS reconciler allowed to manage RepoSyncs but not RoleBindings
		ntopts.RepoSyncPermissions(policy.RepoSyncAdmin()),
		ntopts.NamespaceRepo(nn1.Namespace, nn1.Name),
		ntopts.NamespaceRepo(nn6.Namespace, nn6.Name))

	// Cleanup all unmanaged RepoSyncs BEFORE the root-sync is deleted!
	// Otherwise, the test Namespace will be deleted while still containing
	// RepoSyncs, which could block deletion if their finalizer hangs.
	// This also replaces depends-on deletion ordering (RoleBinding -> RepoSync),
	// which can't be used by unmanaged syncs or objects in different repos.
	nt.T.Cleanup(func() {
		nt.T.Log("[CLEANUP] Deleting test RepoSyncs")
		var rsList []v1beta1.RepoSync
		rsNNs := []types.NamespacedName{nn1, nn2, nn4}
		for _, rsNN := range rsNNs {
			rs := &v1beta1.RepoSync{}
			err := nt.Get(rsNN.Name, rsNN.Namespace, rs)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					nt.T.Error(err)
				}
			} else {
				rsList = append(rsList, *rs)
			}
		}
		if err := nomostest.ResetRepoSyncs(nt, rsList); err != nil {
			nt.T.Error(err)
		}
	})

	var newRepos []types.NamespacedName
	newRepos = append(newRepos, nomostest.RootSyncNN(rr2))
	newRepos = append(newRepos, nomostest.RootSyncNN(rr3))
	newRepos = append(newRepos, nn2)
	newRepos = append(newRepos, nn3)
	newRepos = append(newRepos, nn4)
	newRepos = append(newRepos, nn5)

	if nt.GitProvider.Type() == e2e.Local {
		nomostest.InitGitRepos(nt, newRepos...)
	}
	rr2Repo := nomostest.NewRepository(nt, nomostest.RootRepo, nomostest.RootSyncNN(rr2), filesystem.SourceFormatUnstructured)
	rr3Repo := nomostest.NewRepository(nt, nomostest.RootRepo, nomostest.RootSyncNN(rr3), filesystem.SourceFormatUnstructured)
	nn2Repo := nomostest.NewRepository(nt, nomostest.NamespaceRepo, nn2, filesystem.SourceFormatUnstructured)
	nn3Repo := nomostest.NewRepository(nt, nomostest.NamespaceRepo, nn3, filesystem.SourceFormatUnstructured)
	nn4Repo := nomostest.NewRepository(nt, nomostest.NamespaceRepo, nn4, filesystem.SourceFormatUnstructured)
	nn5Repo := nomostest.NewRepository(nt, nomostest.NamespaceRepo, nn5, filesystem.SourceFormatUnstructured)

	nrb2 := nomostest.RepoSyncRoleBinding(nn2)
	nrb3 := nomostest.RepoSyncRoleBinding(nn3)
	nrb4 := nomostest.RepoSyncRoleBinding(nn4)
	nrb5 := nomostest.RepoSyncRoleBinding(nn5)

	nt.T.Logf("Adding Namespace & RoleBindings for RepoSyncs")
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("acme/cluster/ns-%s.yaml", testNs), fake.NamespaceObject(testNs))
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("acme/namespaces/%s/rb-%s.yaml", testNs, nn2.Name), nrb2)
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("acme/namespaces/%s/rb-%s.yaml", testNs, nn4.Name), nrb4)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Adding Namespace & RoleBindings for RepoSyncs")

	nt.T.Logf("Add RootSync %s to the repository of RootSync %s", rr2, configsync.RootSyncName)
	nt.RootRepos[rr2] = rr2Repo
	rs2 := nomostest.RootSyncObjectV1Alpha1FromRootRepo(nt, rr2)
	rs2ConfigFile := fmt.Sprintf("acme/rootsyncs/%s.yaml", rr2)
	nt.RootRepos[configsync.RootSyncName].Add(rs2ConfigFile, rs2)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Adding RootSync: " + rr2)
	// Wait for all RootSyncs and RepoSyncs to be synced, including the new RootSync rr2.
	nt.WaitForRepoSyncs()

	nt.T.Logf("Add RootSync %s to the repository of RootSync %s", rr3, rr2)
	nt.RootRepos[rr3] = rr3Repo
	rs3 := nomostest.RootSyncObjectV1Alpha1FromRootRepo(nt, rr3)
	rs3ConfigFile := fmt.Sprintf("acme/rootsyncs/%s.yaml", rr3)
	nt.RootRepos[rr2].Add(rs3ConfigFile, rs3)
	nt.RootRepos[rr2].CommitAndPush("Adding RootSync: " + rr3)
	// Wait for all RootSyncs and RepoSyncs to be synced, including the new RootSync rr3.
	nt.WaitForRepoSyncs()

	nt.T.Logf("Create RepoSync %s", nn2)
	nt.NonRootRepos[nn2] = nn2Repo
	nrs2 := nomostest.RepoSyncObjectV1Alpha1FromNonRootRepo(nt, nn2)
	if err := nt.Create(nrs2); err != nil {
		nt.T.Fatal(err)
	}
	// RoleBinding (nrb2) managed by RootSync root-sync, because the namespace
	// tenant does not have permission to manage RBAC.
	// Wait for all RootSyncs and RepoSyncs to be synced, including the new RepoSync nr2.
	nt.WaitForRepoSyncs()

	nt.T.Logf("Add RepoSync %s to RootSync %s", nn3, configsync.RootSyncName)
	nt.NonRootRepos[nn3] = nn3Repo
	nrs3 := nomostest.RepoSyncObjectV1Alpha1FromNonRootRepo(nt, nn3)
	// Ensure the RoleBinding is deleted after the RepoSync
	if err := nomostest.SetDependencies(nrs3, nrb3); err != nil {
		nt.T.Fatal(err)
	}
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("acme/reposyncs/%s.yaml", nn3.Name), nrs3)
	nt.RootRepos[configsync.RootSyncName].Add(fmt.Sprintf("acme/namespaces/%s/rb-%s.yaml", testNs, nn3.Name), nrb3)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Adding RepoSync: " + nn3.String())
	// Wait for all RootSyncs and RepoSyncs to be synced, including the new RepoSync nr3.
	nt.WaitForRepoSyncs()

	nt.T.Logf("Add RepoSync %s to RepoSync %s", nn4, nn2)
	nt.NonRootRepos[nn4] = nn4Repo
	nrs4 := nomostest.RepoSyncObjectV1Alpha1FromNonRootRepo(nt, nn4)
	nt.NonRootRepos[nn2].Add(fmt.Sprintf("acme/reposyncs/%s.yaml", nn4.Name), nrs4)
	// RoleBinding (nrb4) managed by RootSync root-sync, because RepoSync (nr2)
	// does not have permission to manage RBAC.
	nt.NonRootRepos[nn2].CommitAndPush("Adding RepoSync: " + nn4.String())
	// Wait for all RootSyncs and RepoSyncs to be synced, including the new RepoSync nr4.
	nt.WaitForRepoSyncs()

	nt.T.Logf("Add RepoSync %s to RootSync %s", nn5, rr1)
	nt.NonRootRepos[nn5] = nn5Repo
	nrs5 := nomostest.RepoSyncObjectV1Beta1FromNonRootRepo(nt, nn5)
	// Ensure the RoleBinding is deleted after the RepoSync
	if err := nomostest.SetDependencies(nrs5, nrb5); err != nil {
		nt.T.Fatal(err)
	}
	nt.RootRepos[rr1].Add(fmt.Sprintf("acme/reposyncs/%s.yaml", nn5.Name), nrs5)
	nt.RootRepos[rr1].Add(fmt.Sprintf("acme/namespaces/%s/rb-%s.yaml", testNs, nn5.Name), nrb5)
	nt.RootRepos[rr1].CommitAndPush("Adding RepoSync: " + nn5.String())
	// Wait for all RootSyncs and RepoSyncs to be synced, including the new RepoSync nr5.
	nt.WaitForRepoSyncs()

	nt.T.Logf("Validate reconciler Deployment labels")
	validateReconcilerResource(nt, kinds.Deployment(), map[string]string{"app": "reconciler"}, 10)
	validateReconcilerResource(nt, kinds.Deployment(), map[string]string{metadata.SyncNamespaceLabel: configsync.ControllerNamespace}, 4)
	validateReconcilerResource(nt, kinds.Deployment(), map[string]string{metadata.SyncNamespaceLabel: testNs}, 5)
	validateReconcilerResource(nt, kinds.Deployment(), map[string]string{metadata.SyncNamespaceLabel: testNs2}, 1)
	validateReconcilerResource(nt, kinds.Deployment(), map[string]string{metadata.SyncNameLabel: rr1}, 1)
	validateReconcilerResource(nt, kinds.Deployment(), map[string]string{metadata.SyncNameLabel: nr1}, 2)

	validateReconcilerResource(nt, kinds.Pod(), map[string]string{"app": "reconciler"}, 10)
	validateReconcilerResource(nt, kinds.Pod(), map[string]string{metadata.SyncNamespaceLabel: configsync.ControllerNamespace}, 4)
	validateReconcilerResource(nt, kinds.Pod(), map[string]string{metadata.SyncNamespaceLabel: testNs}, 5)
	validateReconcilerResource(nt, kinds.Pod(), map[string]string{metadata.SyncNamespaceLabel: testNs2}, 1)
	validateReconcilerResource(nt, kinds.Pod(), map[string]string{metadata.SyncNameLabel: rr1}, 1)
	validateReconcilerResource(nt, kinds.Pod(), map[string]string{metadata.SyncNameLabel: nr1}, 2)

	validateReconcilerResource(nt, kinds.ServiceAccount(), map[string]string{metadata.SyncNamespaceLabel: configsync.ControllerNamespace}, 4)
	validateReconcilerResource(nt, kinds.ServiceAccount(), map[string]string{metadata.SyncNamespaceLabel: testNs}, 5)
	validateReconcilerResource(nt, kinds.ServiceAccount(), map[string]string{metadata.SyncNamespaceLabel: testNs2}, 1)
	validateReconcilerResource(nt, kinds.ServiceAccount(), map[string]string{metadata.SyncNameLabel: rr1}, 1)
	validateReconcilerResource(nt, kinds.ServiceAccount(), map[string]string{metadata.SyncNameLabel: nr1}, 2)

	// Reconciler-manager doesn't copy the secret of RootSync's secretRef.
	validateReconcilerResource(nt, kinds.Secret(), map[string]string{metadata.SyncNamespaceLabel: configsync.ControllerNamespace}, 0)
	validateReconcilerResource(nt, kinds.Secret(), map[string]string{metadata.SyncNamespaceLabel: testNs}, 5)
	validateReconcilerResource(nt, kinds.Secret(), map[string]string{metadata.SyncNamespaceLabel: testNs2}, 1)
	validateReconcilerResource(nt, kinds.Secret(), map[string]string{metadata.SyncNameLabel: nr1}, 2)
}

func validateReconcilerResource(nt *nomostest.NT, gvk schema.GroupVersionKind, labels map[string]string, expectedCount int) {
	list := &unstructured.UnstructuredList{}
	listGVK := gvk
	listGVK.Kind += "List"
	list.SetGroupVersionKind(listGVK)

	if err := nt.List(list, client.MatchingLabels(labels)); err != nil {
		nt.T.Fatal(err)
	}
	if len(list.Items) != expectedCount {
		nt.T.Fatalf("expected %d reconciler %s(s), got %d", expectedCount, gvk.Kind, len(list.Items))
	}
}

func TestConflictingDefinitions_RootToNamespace(t *testing.T) {
	repoSyncNN := nomostest.RepoSyncNN(testNs, "rs-test")
	nt := nomostest.New(t, nomostesting.MultiRepos,
		ntopts.NamespaceRepo(repoSyncNN.Namespace, repoSyncNN.Name),
		ntopts.RepoSyncPermissions(policy.RBACAdmin()), // NS Reconciler manages Roles
	)

	podRoleFilePath := fmt.Sprintf("acme/namespaces/%s/pod-role.yaml", testNs)
	nt.T.Logf("Add a Role to root: %s", configsync.RootSyncName)
	nt.RootRepos[configsync.RootSyncName].Add(podRoleFilePath, rootPodRole())
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add pod viewer role")
	nt.WaitForRepoSyncs()

	nsReconcilerName := core.NsReconcilerName(repoSyncNN.Namespace, repoSyncNN.Name)
	// Validate multi-repo metrics from root reconciler.
	err := nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName,
			nt.DefaultRootSyncObjectCount()+1, // 1 for the test Role managed by the RootSync
			metrics.ResourceCreated("Role"))
		if err != nil {
			return err
		}
		err = nt.ValidateMultiRepoMetrics(nsReconcilerName,
			0) // 0 for the test Role NOT managed by the RepoSync
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}

	nt.T.Logf("Declare a conflicting Role in the Namespace repo: %s", repoSyncNN)
	nt.NonRootRepos[repoSyncNN].Add(podRoleFilePath, namespacePodRole())
	nt.NonRootRepos[repoSyncNN].CommitAndPush("add conflicting pod owner role")

	nt.T.Logf("The RootSync should report no problems")
	nt.WaitForRepoSyncs(nomostest.RootSyncOnly())

	nt.T.Logf("The RepoSync %s reports a problem since it can't sync the declaration.", repoSyncNN)
	nt.WaitForRepoSyncSyncError(repoSyncNN.Namespace, repoSyncNN.Name, status.ManagementConflictErrorCode, "declared in another repository")

	nt.T.Logf("Validate reconciler error metric is emitted from namespace reconciler %s", repoSyncNN)
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateReconcilerErrors(nsReconcilerName, 0, 1)
	})
	if err != nil {
		nt.T.Error(err)
	}

	nt.T.Logf("Ensure the Role matches the one in the Root repo %s", configsync.RootSyncName)
	err = nt.Validate("pods", testNs, &rbacv1.Role{},
		roleHasRules(rootPodRole().Rules),
		nomostest.IsManagedBy(nt, declared.RootReconciler, configsync.RootSyncName))
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Remove the declaration from the Root repo %s", configsync.RootSyncName)
	nt.RootRepos[configsync.RootSyncName].Remove(podRoleFilePath)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("remove conflicting pod role from Root")
	nt.WaitForRepoSyncs()

	nt.T.Logf("Ensure the Role is updated to the one in the Namespace repo %s", repoSyncNN)
	err = nt.Validate("pods", testNs, &rbacv1.Role{},
		roleHasRules(namespacePodRole().Rules),
		nomostest.IsManagedBy(nt, declared.Scope(repoSyncNN.Namespace), repoSyncNN.Name))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate multi-repo metrics from root reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName,
			nt.DefaultRootSyncObjectCount(), // 0 for the test Role NOT managed by the RootSync
			metrics.ResourceDeleted("Role"))
		if err != nil {
			return err
		}
		err = nt.ValidateMultiRepoMetrics(nsReconcilerName,
			1) // 1 for the test Role managed by the RepoSync
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}
}

func TestConflictingDefinitions_NamespaceToRoot(t *testing.T) {
	repoSyncNN := nomostest.RepoSyncNN(testNs, "rs-test")
	nt := nomostest.New(t, nomostesting.MultiRepos,
		ntopts.NamespaceRepo(repoSyncNN.Namespace, repoSyncNN.Name),
		ntopts.RepoSyncPermissions(policy.RBACAdmin()), // NS reconciler manages Roles
	)

	podRoleFilePath := fmt.Sprintf("acme/namespaces/%s/pod-role.yaml", testNs)
	nt.T.Logf("Add a Role to Namespace repo: %s", configsync.RootSyncName)
	nt.NonRootRepos[repoSyncNN].Add(podRoleFilePath, namespacePodRole())
	nt.NonRootRepos[repoSyncNN].CommitAndPush("declare Role")
	nt.WaitForRepoSyncs()

	err := nt.Validate("pods", testNs, &rbacv1.Role{},
		roleHasRules(namespacePodRole().Rules),
		nomostest.IsManagedBy(nt, declared.Scope(repoSyncNN.Namespace), repoSyncNN.Name))
	if err != nil {
		nt.T.Fatal(err)
	}

	nsReconcilerName := core.NsReconcilerName(repoSyncNN.Namespace, repoSyncNN.Name)
	// Validate multi-repo metrics from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nsReconcilerName,
			1, // 1 for the test Role managed by the RepoSync
			metrics.ResourceCreated("Role"))
		if err != nil {
			return err
		}
		err = nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName,
			nt.DefaultRootSyncObjectCount()) // 0 for the test Role NOT managed by the RootSync
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}

	nt.T.Logf("Declare a conflicting Role in the Root repo: %s", configsync.RootSyncName)
	nt.RootRepos[configsync.RootSyncName].Add(podRoleFilePath, rootPodRole())
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add conflicting pod role to Root")

	nt.T.Logf("The RootSync should update the Role")
	nt.WaitForRepoSyncs(nomostest.RootSyncOnly())
	nt.T.Logf("The RepoSync %s reports a problem since it can't sync the declaration.", testNs)
	nt.WaitForRepoSyncSyncError(repoSyncNN.Namespace, repoSyncNN.Name, status.ManagementConflictErrorCode, "declared in another repository")

	// Validate reconciler error metric is emitted from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateReconcilerErrors(nsReconcilerName, 0, 1)
	})
	if err != nil {
		nt.T.Error(err)
	}

	nt.T.Logf("Ensure the Role matches the one in the Root repo %s", configsync.RootSyncName)
	err = nt.Validate("pods", testNs, &rbacv1.Role{},
		roleHasRules(rootPodRole().Rules),
		nomostest.IsManagedBy(nt, declared.RootReconciler, configsync.RootSyncName))
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Remove the Role from the Namespace repo %s", repoSyncNN)
	nt.NonRootRepos[repoSyncNN].Remove(podRoleFilePath)
	nt.NonRootRepos[repoSyncNN].CommitAndPush("remove conflicting pod role from Namespace repo")
	nt.WaitForRepoSyncs()

	nt.T.Logf("Ensure the Role still matches the one in the Root repo %s", configsync.RootSyncName)
	err = nt.Validate("pods", testNs, &rbacv1.Role{},
		roleHasRules(rootPodRole().Rules),
		nomostest.IsManagedBy(nt, declared.RootReconciler, configsync.RootSyncName))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate multi-repo metrics from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nsReconcilerName,
			0) // 0 for the test Role NOT managed by the RepoSync
		if err != nil {
			return err
		}
		err = nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName,
			nt.DefaultRootSyncObjectCount()+1) // 1 for the test Role managed by the RootSync
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}
}

func TestConflictingDefinitions_RootToRoot(t *testing.T) {
	rootSync2 := "root-test"
	// If declaring RootSync in a Root repo, the source format has to be unstructured.
	// Otherwise, the hierarchical validator will complain that the config-management-system has configs but missing a Namespace config.
	nt := nomostest.New(t, nomostesting.MultiRepos, ntopts.Unstructured, ntopts.RootRepo(rootSync2))

	podRoleFilePath := fmt.Sprintf("acme/namespaces/%s/pod-role.yaml", testNs)
	nt.T.Logf("Add a Role to root: %s", configsync.RootSyncName)
	nt.RootRepos[configsync.RootSyncName].Add(podRoleFilePath, rootPodRole())
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add pod viewer role")
	nt.WaitForRepoSyncs()
	nt.T.Logf("Ensure the Role is managed by Root %s", configsync.RootSyncName)
	role := &rbacv1.Role{}
	err := nt.Validate("pods", testNs, role,
		roleHasRules(rootPodRole().Rules),
		nomostest.IsManagedBy(nt, declared.RootReconciler, configsync.RootSyncName))
	if err != nil {
		nt.T.Fatal(err)
	}

	roleResourceVersion := role.ResourceVersion

	nt.T.Logf("Declare a conflicting Role in another Root repo: %s", rootSync2)
	nt.RootRepos[rootSync2].Add(podRoleFilePath, rootPodRole())
	nt.RootRepos[rootSync2].CommitAndPush("add conflicting pod owner role")

	// When the webhook is enabled, it will block adoption of managed objects.
	nt.T.Logf("Both RootSyncs should still report conflicts with the webhook enabled")
	tg := taskgroup.New()
	// Reconciler conflict, detected by the second reconciler & reported to the first reconciler's RootSync
	tg.Go(func() error {
		return nomostest.WatchObject(nt, kinds.RootSyncV1Beta1(), configsync.RootSyncName, configsync.ControllerNamespace,
			[]nomostest.Predicate{
				nomostest.RootSyncHasSyncError(nt, status.ManagementConflictErrorCode, "declared in another repository"),
			})
	})
	// Reconciler conflict, detected by the second reconciler
	tg.Go(func() error {
		return nomostest.WatchObject(nt, kinds.RootSyncV1Beta1(), rootSync2, configsync.ControllerNamespace,
			[]nomostest.Predicate{
				nomostest.RootSyncHasSyncError(nt, status.ManagementConflictErrorCode, "declared in another repository"),
			})
	})
	// Webhook rejection detected by the second reconciler's applier
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/admission/plugin/webhook/errors/statuserror.go#L29
	tg.Go(func() error {
		return nomostest.WatchObject(nt, kinds.RootSyncV1Beta1(), rootSync2, configsync.ControllerNamespace,
			[]nomostest.Predicate{
				nomostest.RootSyncHasSyncError(nt, applier.ApplierErrorCode, "denied the request"),
			})
	})
	if err := tg.Wait(); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("The Role resource version should not be changed")
	if err := nt.Validate("pods", testNs, &rbacv1.Role{},
		nomostest.ResourceVersionEquals(nt, roleResourceVersion)); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Stop the admission webhook, the remediator should report the conflicts")
	nomostest.StopWebhook(nt)
	nt.T.Logf("The Role resource version should be changed because two reconcilers are fighting with each other")
	err = nomostest.WatchObject(nt, kinds.Role(), "pods", testNs,
		[]nomostest.Predicate{nomostest.ResourceVersionNotEquals(nt, roleResourceVersion)},
		nomostest.WatchTimeout(90*time.Second))
	if err != nil {
		nt.T.Fatal(err)
	}

	// When the webhook is disabled, both RootSyncs will repeatedly try to adopt the object.
	nt.T.Logf("Both RootSyncs should still report conflicts with the webhook disabled")
	tg = taskgroup.New()
	// Reconciler conflict, detected by the first reconciler's applier OR reported by the second reconciler
	tg.Go(func() error {
		return nomostest.WatchObject(nt, kinds.RootSyncV1Beta1(), configsync.RootSyncName, configsync.ControllerNamespace,
			[]nomostest.Predicate{
				nomostest.RootSyncHasSyncError(nt, status.ManagementConflictErrorCode, "declared in another repository"),
			})
	})
	// Reconciler conflict, detected by the second reconciler's applier OR reported by the first reconciler
	tg.Go(func() error {
		return nomostest.WatchObject(nt, kinds.RootSyncV1Beta1(), rootSync2, configsync.ControllerNamespace,
			[]nomostest.Predicate{
				nomostest.RootSyncHasSyncError(nt, status.ManagementConflictErrorCode, "declared in another repository"),
			})
	})
	if err := tg.Wait(); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Remove the declaration from one Root repo %s", configsync.RootSyncName)
	nt.RootRepos[configsync.RootSyncName].Remove(podRoleFilePath)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("remove conflicting pod role from Root")
	nt.WaitForRepoSyncs()

	nt.T.Logf("Ensure the Role is managed by the other Root repo %s", rootSync2)
	// The pod role may be deleted from the cluster after it was removed from the `root-sync` Root repo.
	// Therefore, we need to retry here to wait until the `root-test` Root repo recreates the pod role.
	err = nomostest.WatchObject(nt, kinds.Role(), "pods", testNs,
		[]nomostest.Predicate{
			roleHasRules(rootPodRole().Rules),
			nomostest.IsManagedBy(nt, declared.RootReconciler, rootSync2),
		},
		nomostest.WatchTimeout(90*time.Second))
	if err != nil {
		nt.T.Fatal(err)
	}
}

func TestConflictingDefinitions_NamespaceToNamespace(t *testing.T) {
	repoSyncNN1 := nomostest.RepoSyncNN(testNs, "rs-test-1")
	repoSyncNN2 := nomostest.RepoSyncNN(testNs, "rs-test-2")

	nt := nomostest.New(t, nomostesting.MultiRepos,
		ntopts.RepoSyncPermissions(policy.RBACAdmin()), // NS reconciler manages Roles
		ntopts.NamespaceRepo(repoSyncNN1.Namespace, repoSyncNN1.Name),
		ntopts.NamespaceRepo(repoSyncNN2.Namespace, repoSyncNN2.Name))

	podRoleFilePath := fmt.Sprintf("acme/namespaces/%s/pod-role.yaml", testNs)
	nt.T.Logf("Add a Role to Namespace: %s", repoSyncNN1)
	nt.NonRootRepos[repoSyncNN1].Add(podRoleFilePath, namespacePodRole())
	nt.NonRootRepos[repoSyncNN1].CommitAndPush("add pod viewer role")
	nt.WaitForRepoSyncs()
	role := &rbacv1.Role{}
	nt.T.Logf("Ensure the Role is managed by Namespace Repo %s", repoSyncNN1)
	err := nt.Validate("pods", testNs, role,
		roleHasRules(namespacePodRole().Rules),
		nomostest.IsManagedBy(nt, declared.Scope(repoSyncNN1.Namespace), repoSyncNN1.Name))
	if err != nil {
		nt.T.Fatal(err)
	}
	roleResourceVersion := role.ResourceVersion

	// Validate multi-repo metrics from namespace reconciler.
	nsReconcilerName1 := core.NsReconcilerName(repoSyncNN1.Namespace, repoSyncNN1.Name)
	nsReconcilerName2 := core.NsReconcilerName(repoSyncNN2.Namespace, repoSyncNN2.Name)
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nsReconcilerName1,
			1) // 1 for the test Role managed by the 1st RepoSync
		if err != nil {
			return err
		}
		err = nt.ValidateMultiRepoMetrics(nsReconcilerName2,
			0) // 0 for the test Role NOT managed by the 2st RepoSync
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}

	nt.T.Logf("Declare a conflicting Role in another Namespace repo: %s", repoSyncNN2)
	nt.NonRootRepos[repoSyncNN2].Add(podRoleFilePath, namespacePodRole())
	nt.NonRootRepos[repoSyncNN2].CommitAndPush("add conflicting pod owner role")

	nt.T.Logf("Only RepoSync %s reports the conflict error because kpt_applier won't update the resource", repoSyncNN2)
	nt.WaitForRepoSyncSyncError(repoSyncNN2.Namespace, repoSyncNN2.Name, status.ManagementConflictErrorCode, "declared in another repository")
	nt.WaitForSync(kinds.RepoSyncV1Beta1(), repoSyncNN1.Name, repoSyncNN1.Namespace,
		nt.DefaultWaitTimeout, nomostest.DefaultRepoSha1Fn, nomostest.RepoSyncHasStatusSyncCommit, nil)
	nt.T.Logf("The Role resource version should not be changed")
	err = nt.Validate("pods", testNs, &rbacv1.Role{},
		nomostest.ResourceVersionEquals(nt, roleResourceVersion))
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Stop the admission webhook, the remediator should not be affected, which still reports the conflicts")
	nomostest.StopWebhook(nt)
	nt.WaitForRepoSyncSyncError(repoSyncNN2.Namespace, repoSyncNN2.Name, status.ManagementConflictErrorCode, "declared in another repository")
	nt.WaitForSync(kinds.RepoSyncV1Beta1(), repoSyncNN1.Name, repoSyncNN1.Namespace,
		nt.DefaultWaitTimeout, nomostest.DefaultRepoSha1Fn, nomostest.RepoSyncHasStatusSyncCommit, nil)

	nt.T.Logf("Validate reconciler error metric is emitted from Namespace reconciler %s", repoSyncNN2)
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateReconcilerErrors(nsReconcilerName2, 0, 1)
	})
	if err != nil {
		nt.T.Error(err)
	}

	nt.T.Logf("Remove the declaration from one Namespace repo %s", repoSyncNN1)
	nt.NonRootRepos[repoSyncNN1].Remove(podRoleFilePath)
	nt.NonRootRepos[repoSyncNN1].CommitAndPush("remove conflicting pod role from Namespace")
	nt.WaitForRepoSyncs()

	nt.T.Logf("Ensure the Role is managed by the other Namespace repo %s", repoSyncNN2)
	err = nt.Validate("pods", testNs, &rbacv1.Role{},
		roleHasRules(namespacePodRole().Rules),
		nomostest.IsManagedBy(nt, declared.Scope(repoSyncNN2.Namespace), repoSyncNN2.Name))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate multi-repo metrics from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nsReconcilerName1,
			0) // 0 for the test Role NOT managed by the 1st RepoSync
		if err != nil {
			return err
		}
		err = nt.ValidateMultiRepoMetrics(nsReconcilerName2,
			1) // 1 for the test Role managed by the 2st RepoSync
		if err != nil {
			return err
		}
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		nt.T.Error(err)
	}
}

func TestControllerValidationErrors(t *testing.T) {
	nt := nomostest.New(t, nomostesting.MultiRepos)

	testNamespace := fake.NamespaceObject(testNs)
	if err := nt.Create(testNamespace); err != nil {
		nt.T.Fatal(err)
	}
	t.Cleanup(func() {
		if err := nt.Delete(testNamespace); err != nil {
			nt.T.Fatal(err)
		}
	})

	rootSync := &v1beta1.RootSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-test",
			Namespace: testNs,
		},
		Spec: v1beta1.RootSyncSpec{
			Git: &v1beta1.Git{
				Auth: "none",
			},
		},
	}
	if err := nt.Create(rootSync); err != nil {
		nt.T.Fatal(err)
	}
	nt.WaitForRootSyncStalledError(rootSync.Namespace, rootSync.Name, "Validation", "RootSync objects are only allowed in the config-management-system namespace, not in test-ns")
	t.Cleanup(func() {
		if err := nt.Delete(rootSync); err != nil {
			nt.T.Fatal(err)
		}
	})

	nnControllerNamespace := nomostest.RepoSyncNN(configsync.ControllerNamespace, configsync.RepoSyncName)
	rs := nomostest.RepoSyncObjectV1Beta1(nnControllerNamespace, "", filesystem.SourceFormatUnstructured)
	if err := nt.Create(rs); err != nil {
		nt.T.Fatal(err)
	}
	nt.WaitForRepoSyncStalledError(rs.Namespace, rs.Name, "Validation", "RepoSync objects are not allowed in the config-management-system namespace")
	if err := nt.Delete(rs); err != nil {
		nt.T.Fatal(err)
	}

	longBytes := make([]byte, validation.DNS1123SubdomainMaxLength)
	for i := range longBytes {
		longBytes[i] = 'a'
	}
	veryLongName := string(longBytes)
	nnTooLong := nomostest.RepoSyncNN(testNs, veryLongName)
	rs = nomostest.RepoSyncObjectV1Beta1(nnTooLong, "https://github.com/test/test", filesystem.SourceFormatUnstructured)
	if err := nt.Create(rs); err != nil {
		nt.T.Fatal(err)
	}
	nt.WaitForRepoSyncStalledError(rs.Namespace, rs.Name, "Validation",
		fmt.Sprintf(`Invalid reconciler name "ns-reconciler-%s-%s-%d": must be no more than %d characters.`,
			testNs, veryLongName, len(veryLongName), validation.DNS1123SubdomainMaxLength))
	t.Cleanup(func() {
		if err := nt.Delete(rs); err != nil {
			nt.T.Fatal(err)
		}
	})

	nnInvalidSecretRef := nomostest.RepoSyncNN(testNs, "repo-test")
	rsInvalidSecretRef := nomostest.RepoSyncObjectV1Beta1(nnInvalidSecretRef, "https://github.com/test/test", filesystem.SourceFormatUnstructured)
	rsInvalidSecretRef.Spec.SecretRef = &v1beta1.SecretReference{Name: veryLongName}
	if err := nt.Create(rsInvalidSecretRef); err != nil {
		nt.T.Fatal(err)
	}
	nt.WaitForRepoSyncStalledError(rsInvalidSecretRef.Namespace, rsInvalidSecretRef.Name, "Validation",
		fmt.Sprintf(`The managed secret name "ns-reconciler-%s-%s-%d-%s" is invalid: must be no more than %d characters. To fix it, update '.spec.git.secretRef.name'`,
			testNs, rsInvalidSecretRef.Name, len(rsInvalidSecretRef.Name), v1beta1.GetSecretName(rsInvalidSecretRef.Spec.SecretRef), validation.DNS1123SubdomainMaxLength))
	t.Cleanup(func() {
		if err := nt.Delete(rsInvalidSecretRef); err != nil {
			nt.T.Fatal(err)
		}
	})
}

func rootPodRole() *rbacv1.Role {
	result := fake.RoleObject(
		core.Name("pods"),
		core.Namespace(testNs),
	)
	result.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list"},
		},
	}
	return result
}

func namespacePodRole() *rbacv1.Role {
	result := fake.RoleObject(
		core.Name("pods"),
		core.Namespace(testNs),
	)
	result.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{corev1.GroupName},
			Resources: []string{"pods"},
			Verbs:     []string{"*"},
		},
	}
	return result
}

func roleHasRules(wantRules []rbacv1.PolicyRule) nomostest.Predicate {
	return func(o client.Object) error {
		r, isRole := o.(*rbacv1.Role)
		if !isRole {
			return nomostest.WrongTypeErr(o, &rbacv1.Role{})
		}

		if diff := cmp.Diff(wantRules, r.Rules); diff != "" {
			return errors.Errorf("Pod Role .rules diff: %s", diff)
		}
		return nil
	}
}
