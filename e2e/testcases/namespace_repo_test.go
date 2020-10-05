package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestNamespaceRepo_Centralized(t *testing.T) {
	bsNamespace := "bookstore"

	nt := nomostest.New(
		t,
		ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(bsNamespace),
		ntopts.WithCentralizedControl,
	)

	repo, exist := nt.NonRootRepos[bsNamespace]
	if !exist {
		t.Fatal("nonexistent repo")
	}

	// Validate service account 'store' not present.
	err := nt.ValidateNotFound("store", bsNamespace, &corev1.ServiceAccount{})
	if err != nil {
		t.Errorf("store service account already present: %v", err)
	}

	sa := fake.ServiceAccountObject("store", core.Namespace(bsNamespace))
	repo.Add("acme/sa.yaml", sa)
	repo.CommitAndPush("Adding service account")
	nt.WaitForRepoSyncs()

	// Validate service account 'store' is present.
	_, err = nomostest.Retry(15*time.Second, func() error {
		return nt.Validate("store", bsNamespace, &corev1.ServiceAccount{})
	})
	if err != nil {
		t.Fatalf("service account store not found: %v", err)
	}
}

func TestNamespaceRepo_Delegated(t *testing.T) {
	bsNamespaceRepo := "bookstore"

	nt := nomostest.New(
		t,
		ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(bsNamespaceRepo),
		ntopts.WithDelegatedControl,
	)

	repo, exist := nt.NonRootRepos[bsNamespaceRepo]
	if !exist {
		t.Fatal("nonexistent repo")
	}

	// Validate service account 'store' not present.
	err := nt.ValidateNotFound("store", bsNamespaceRepo, &corev1.ServiceAccount{})
	if err != nil {
		t.Errorf("store service account already present: %v", err)
	}

	sa := fake.ServiceAccountObject("store", core.Namespace(bsNamespaceRepo))
	repo.Add("acme/sa.yaml", sa)
	repo.CommitAndPush("Adding service account")
	nt.WaitForRepoSyncs()

	// Validate service account 'store' is present.
	err = nt.Validate("store", bsNamespaceRepo, &corev1.ServiceAccount{})
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteRepoSync_Delegated(t *testing.T) {
	bsNamespace := "bookstore"

	nt := nomostest.New(
		t,
		ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(bsNamespace),
		ntopts.WithDelegatedControl,
	)

	var rs v1alpha1.RepoSync
	if err := nt.Get(v1alpha1.RepoSyncName, bsNamespace, &rs); err != nil {
		t.Fatal(err)
	}

	// Delete RepoSync custom resource from the cluster.
	err := nt.Delete(&rs)
	if err != nil {
		t.Fatalf("RepoSync delete failed: %v", err)
	}

	checkRepoSyncResourcesNotPresent(bsNamespace, nt)
}

func TestDeleteRepoSync_Centralized(t *testing.T) {
	bsNamespace := "bookstore"

	nt := nomostest.New(
		t,
		ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(bsNamespace),
		ntopts.WithCentralizedControl,
	)

	// Remove RepoSync resource from Root Repository.
	nt.Root.Remove(nomostest.StructuredNSPath(bsNamespace, nomostest.RepoSyncFileName))
	nt.Root.CommitAndPush("Removing RepoSync from the Root Repository")
	// Remove from NamespaceRepos so we don't try to check that it is syncing,
	// as we've just deleted it.
	delete(nt.NamespaceRepos, bsNamespace)
	nt.WaitForRepoSyncs()

	checkRepoSyncResourcesNotPresent(bsNamespace, nt)
}

func checkRepoSyncResourcesNotPresent(namespace string, nt *nomostest.NT) {
	_, err := nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(v1alpha1.RepoSyncName, namespace, fake.RepoSyncObject())
	})
	if err != nil {
		nt.T.Errorf("RepoSync present after deletion: %v", err)
	}

	// Verify Namespace Reconciler deployment no longer present.
	_, err = nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(fmt.Sprintf("%s-%s", "ns-reconciler", namespace), v1.NSConfigManagementSystem, fake.DeploymentObject())
	})
	if err != nil {
		nt.T.Errorf("Reconciler deployment present after deletion: %v", err)
	}

	// validate Namespace Reconciler configmaps are no longer present.
	err1 := nt.ValidateNotFound("ns-reconciler-bookstore-git-sync", v1.NSConfigManagementSystem, fake.ConfigMapObject())
	err2 := nt.ValidateNotFound("ns-reconciler-bookstore-reconciler", v1.NSConfigManagementSystem, fake.ConfigMapObject())
	if err1 != nil || err2 != nil {
		if err1 != nil {
			nt.T.Error(err1)
		}
		if err2 != nil {
			nt.T.Error(err2)
		}
		nt.T.FailNow()
	}
}
