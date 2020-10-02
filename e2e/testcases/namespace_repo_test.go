package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
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
