package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

func TestNamespaceRepo_Centralized(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	rs := fake.RepoSyncObject(core.Namespace("foo"))
	rs.Spec.Repo = "TODO(b/168915318)"
	rs.Spec.Auth = "none"

	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/repo-sync.yaml", rs)
	nt.Root.CommitAndPush("Adding foo namespace and RepoSync")
	nt.WaitForRepoSyncs()

	err := nt.Validate(rs.Name, "foo", &v1alpha1.RepoSync{})
	if err != nil {
		t.Error(err)
	}

	// TODO(b/168915318): Validate that the objects in the namespace repo get synced.
}

func TestNamespaceRepo_Delegated(t *testing.T) {
	bsNamespaceRepo := "bookstore"

	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.NamespaceRepo(bsNamespaceRepo))

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
