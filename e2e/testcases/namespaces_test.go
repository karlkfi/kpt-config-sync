package e2e

import (
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
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

func TestDontDeleteAllNamespaces(t *testing.T) {
	nt := nomostest.New(t)

	// Test Setup + Preconditions.
	// Declare two Namespaces.
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/bar/ns.yaml", fake.NamespaceObject("bar"))
	nt.Root.CommitAndPush("declare multiple Namespaces")
	nt.WaitForRepoSyncs()

	err := nt.Validate("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("bar", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}

	// Remove the only two declared Namespaces.
	// We expect this to fail.
	nt.Root.Remove("acme/namespaces/foo/ns.yaml")
	nt.Root.Remove("acme/namespaces/bar/ns.yaml")
	nt.Root.CommitAndPush("undeclare all Namespaces")

	if nt.MultiRepo {
		_, err = nomostest.Retry(60*time.Second, func() error {
			return nt.Validate("root-sync", "config-management-system",
				&v1alpha1.RootSync{}, rootSyncHasErrors(status.EmptySourceErrorCode))
		})
	} else {
		_, err = nomostest.Retry(60*time.Second, func() error {
			return nt.Validate("repo", "",
				&v1.Repo{}, repoHasErrors("KNV"+status.EmptySourceErrorCode))
		})
	}
	if err != nil {
		// Fail since we needn't continue the test if this action wasn't blocked.
		t.Fatal(err)
	}

	err = nt.Validate("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("bar", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}

	// Add foo back so we resume syncing.
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.CommitAndPush("re-declare foo Namespace")
	nt.WaitForRepoSyncs()

	err = nt.Validate("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = nomostest.Retry(10*time.Second, func() error {
		// It takes a few seconds for Namespaces to terminate.
		return nt.ValidateNotFound("bar", "", &corev1.Namespace{})
	})
	if err != nil {
		t.Fatal(err)
	}

	// Undeclare foo. We expect this to succeed since the user unambiguously wants
	// all Namespaces to be removed.
	nt.Root.Remove("acme/namespaces/foo/ns.yaml")
	nt.Root.CommitAndPush("undeclare foo Namespace")
	nt.WaitForRepoSyncs()

	_, err = nomostest.Retry(10*time.Second, func() error {
		// It takes a few seconds for Namespaces to terminate.
		return nt.ValidateNotFound("foo", "", &corev1.Namespace{})
	})
	if err != nil {
		t.Fatal(err)
	}
	err = nt.ValidateNotFound("bar", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}
}

func rootSyncHasErrors(wantCodes ...string) nomostest.Predicate {
	sort.Strings(wantCodes)

	var wantErrs []v1alpha1.ConfigSyncError
	for _, code := range wantCodes {
		wantErrs = append(wantErrs, v1alpha1.ConfigSyncError{Code: code})
	}

	return func(o core.Object) error {
		rs, isRootSync := o.(*v1alpha1.RootSync)
		if !isRootSync {
			return nomostest.WrongTypeErr(o, &v1alpha1.RootSync{})
		}

		gotErrs := rs.Status.Sync.Errors
		sort.Slice(gotErrs, func(i, j int) bool {
			return gotErrs[i].Code < gotErrs[j].Code
		})

		if diff := cmp.Diff(wantErrs, gotErrs,
			cmpopts.IgnoreFields(v1alpha1.ConfigSyncError{}, "ErrorMessage")); diff != "" {
			return errors.New(diff)
		}
		return nil
	}
}

func repoHasErrors(wantCodes ...string) nomostest.Predicate {
	sort.Strings(wantCodes)

	var wantErrs []v1.ConfigManagementError
	for _, code := range wantCodes {
		wantErrs = append(wantErrs, v1.ConfigManagementError{Code: code})
	}

	return func(o core.Object) error {
		repo, isRepo := o.(*v1.Repo)
		if !isRepo {
			return nomostest.WrongTypeErr(o, &v1.Repo{})
		}

		gotErrs := repo.Status.Source.Errors
		sort.Slice(gotErrs, func(i, j int) bool {
			return gotErrs[i].Code < gotErrs[j].Code
		})

		if diff := cmp.Diff(wantErrs, gotErrs,
			cmpopts.IgnoreFields(v1.ConfigManagementError{}, "ErrorMessage")); diff != "" {
			return errors.New(diff)
		}
		return nil
	}
}
