package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// TestDeclareNamespace runs a test that ensures ACM syncs Namespaces to clusters.
func TestDeclareNamespace(t *testing.T) {
	nt := nomostest.New(t)

	err := nt.ValidateNotFound("foo", "", &corev1.Namespace{})
	if err != nil {
		// Failed test precondition.
		t.Fatal(err)
	}

	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.CommitAndPush("add Namespace")
	nt.WaitForRepoSyncs()

	// Test that the Namespace "foo" exists.
	err = nt.Validate("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Error(err)
	}
}

func TestDeclareImplicitNamespace(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	err := nt.ValidateNotFound("shipping", "", &corev1.Namespace{})
	if err != nil {
		// Failed test precondition. We want to ensure we create the Namespace.
		t.Fatal(err)
	}

	// Phase 1: Declare a Role in a Namespace that doesn't exist, and ensure it
	// gets created.
	nt.Root.Add("acme/role.yaml", fake.RoleObject(core.Name("admin"),
		core.Namespace("shipping")))
	nt.Root.CommitAndPush("add Role in implicit Namespace")
	nt.WaitForRepoSyncs()

	err = nt.Validate("shipping", "", &corev1.Namespace{})
	if err != nil {
		// No need to continue test since Namespace was never created.
		t.Fatal(err)
	}
	err = nt.Validate("admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Error(err)
	}

	// Phase 2: Remove the Role, and ensure the implicit Namespace is NOT deleted.
	nt.Root.Remove("acme/role.yaml")
	nt.Root.CommitAndPush("remove Role")
	nt.WaitForRepoSyncs()

	err = nt.Validate("shipping", "", &corev1.Namespace{})
	if err != nil {
		t.Error(err)
	}
	err = nt.ValidateNotFound("admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Error(err)
	}
}
