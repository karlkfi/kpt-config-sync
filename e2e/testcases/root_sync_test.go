package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDeleteRootSync(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	var rs v1alpha1.RootSync
	err := nt.Validate(v1alpha1.RootSyncName, v1.NSConfigManagementSystem, &rs)
	if err != nil {
		t.Fatal(err)
	}

	// Delete RootSync custom resource from the cluster.
	err = nt.Delete(&rs)
	if err != nil {
		t.Fatalf("deleting RootSync: %v", err)
	}

	_, err = nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(v1alpha1.RootSyncName, v1.NSConfigManagementSystem, fake.RootSyncObject())
	})
	if err != nil {
		t.Errorf("RootSync present after deletion: %v", err)
	}

	// Verify Root Reconciler deployment no longer present.
	_, err = nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound("root-reconciler", v1.NSConfigManagementSystem, fake.DeploymentObject())
	})
	if err != nil {
		t.Errorf("RootSync present after deletion: %v", err)
	}

	// validate Root Reconcoiler configmaps are no longer present.
	err1 := nt.ValidateNotFound("root-reconciler-git-sync", v1.NSConfigManagementSystem, fake.ConfigMapObject())
	err2 := nt.ValidateNotFound("root-reconciler-reconciler", v1.NSConfigManagementSystem, fake.ConfigMapObject())
	err3 := nt.ValidateNotFound("root-reconciler-source-format", v1.NSConfigManagementSystem, fake.ConfigMapObject())
	if err1 != nil || err2 != nil || err3 != nil {
		if err1 != nil {
			t.Error(err1)
		}
		if err2 != nil {
			t.Error(err2)
		}
		if err3 != nil {
			t.Error(err3)
		}
		t.FailNow()
	}
}

func TestUpdateRootSyncGitDirectory(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	// Validate RootSync is present.
	var rs v1alpha1.RootSync
	err := nt.Validate(v1alpha1.RootSyncName, v1.NSConfigManagementSystem, &rs)
	if err != nil {
		t.Fatal(err)
	}

	// Add audit namespace in policy directory acme.
	acmeNS := "audit"
	nt.Root.Add(fmt.Sprintf("%s/namespaces/%s/ns.yaml", rs.Spec.Git.Dir, acmeNS),
		fake.NamespaceObject(acmeNS))

	// Add namespace in policy directory 'foo'.
	fooDir := "foo"
	fooNS := "shipping"
	nt.Root.Add(fmt.Sprintf("%s/namespaces/%s/ns.yaml", fooDir, fooNS),
		fake.NamespaceObject(fooNS))

	// Add repo resource in policy directory 'foo'.
	nt.Root.Add(fmt.Sprintf("%s/system/repo.yaml", fooDir),
		fake.RepoObject())

	nt.Root.CommitAndPush("add namespace to acme directory")
	nt.WaitForRepoSyncs()

	// Validate namespace 'audit' created.
	err = nt.Validate(acmeNS, "", fake.NamespaceObject(acmeNS))
	if err != nil {
		t.Error(err)
	}

	// Validate namespace 'shipping' not present.
	err = nt.ValidateNotFound(fooNS, "", fake.NamespaceObject(fooNS))
	if err != nil {
		t.Errorf("%s present after deletion: %v", fooNS, err)
	}

	// Update RootSync.
	//
	// Get RootSync and then perform Update to avoid update failures due to
	// version mismatch.
	_, err = nomostest.Retry(5*time.Second, func() error {
		rootsync := &v1alpha1.RootSync{}
		err := nt.Get(v1alpha1.RootSyncName, v1.NSConfigManagementSystem, rootsync)
		if err != nil {
			return err
		}

		// Update the policy directory in RootSync Custom Resource.
		rootsync.Spec.Git.Dir = fooDir

		err = nt.Update(rootsync)
		return err
	})
	if err != nil {
		t.Fatalf("RootSync update failed: %v", err)
	}

	// TODO(b/169195578) Validate Source path annotation with policyDir.

	// Validate namespace 'shipping' created.
	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate(fooNS, "", fake.NamespaceObject(fooNS))
	})
	if err != nil {
		t.Error(err)
	}

	// Validate namespace 'audit' no longer present.
	_, err = nomostest.Retry(20*time.Second, func() error {
		return nt.ValidateNotFound(acmeNS, "", fake.NamespaceObject(acmeNS))
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateRootSyncGitBranch(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	// Add audit namespace.
	auditNS := "audit"
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", auditNS),
		fake.NamespaceObject(auditNS))
	nt.Root.CommitAndPush("add namespace to acme directory")
	nt.WaitForRepoSyncs()

	// Validate namespace 'acme' created.
	err := nt.Validate(auditNS, "", fake.NamespaceObject(auditNS))
	if err != nil {
		t.Error(err)
	}

	testBranch := "test-branch"
	testNS := "audit-test"

	// Add a 'test-branch' branch with 'audit-test' namespace.
	nt.Root.CreateBranch(testBranch)
	nt.Root.CheckoutBranch(testBranch)
	nt.Root.Add(fmt.Sprintf("acme/namespaces/%s/ns.yaml", testNS),
		fake.NamespaceObject(testNS))
	nt.Root.CommitAndPushBranch("add audit-test to acme directory", testBranch)

	// Validate namespace 'audit-test' not present to vaidate rootsync is not syncing
	// from 'test-branch' yet.
	err = nt.ValidateNotFound(testNS, "", fake.NamespaceObject(testNS))
	if err != nil {
		t.Errorf("%s present: %v", testNS, err)
	}

	// Update RootSync.
	//
	// Get RootSync and then perform Update.
	rootsync := &v1alpha1.RootSync{}
	err = nt.Get(v1alpha1.RootSyncName, v1.NSConfigManagementSystem, rootsync)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Update the branch in RootSync Custom Resource.
	rootsync.Spec.Git.Branch = testBranch

	err = nt.Update(rootsync)
	if err != nil {
		t.Fatalf("%v", err)
	}
	nt.WaitForRepoSyncs()

	// Validate namespace 'audit-test' created after updating rootsync.
	err = nt.Validate(testNS, "", fake.NamespaceObject(testNS))
	if err != nil {
		t.Error(err)
	}

	// Get RootSync and then perform Update.
	rs := &v1alpha1.RootSync{}
	err = nt.Get(v1alpha1.RootSyncName, v1.NSConfigManagementSystem, rs)
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Switch back to 'main' branch.
	rs.Spec.Git.Branch = nomostest.MainBranch

	err = nt.Update(rs)
	if err != nil {
		t.Fatalf("%v", err)
	}
	nt.WaitForRepoSyncs()

	// Validate namespace 'acme' present.
	err = nt.Validate(auditNS, "", fake.NamespaceObject(auditNS))
	if err != nil {
		t.Error(err)
	}

	// Validate namespace 'audit-test' not present to vaidate rootsync is not syncing
	// from 'test-branch' anymore.
	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.ValidateNotFound(testNS, "", fake.NamespaceObject(testNS))
	})
	if err != nil {
		t.Fatalf("RootSync update failed: %v", err)
	}
}

func TestInvalidRootSyncBranchStatus(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	// Update RootSync to invalid branch name
	rs := fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"branch": "invalid-branch"}}}`)

	// Check for error code 2001 (this is a generic error code for the curent impl, this may change if we
	// make better git error reporting.
	nt.WaitForRootSyncSourceErrorCode("2001")

	// Update RootSync to valid branch name
	rs = fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"branch": "main"}}}`)

	nt.WaitForRootSyncSourceErrorClear()
}

func TestForceRevert(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	nt.Root.Remove("acme/system/repo.yaml")
	nt.Root.CommitAndPush("Cause source error")

	nt.WaitForRootSyncSourceErrorCode("1017")

	nt.Root.Git("reset", "--hard", "HEAD^")
	nt.Root.Git("push", "-f", "origin", "main")

	nt.WaitForRootSyncSourceErrorClear()
}
