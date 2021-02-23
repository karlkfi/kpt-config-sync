package e2e

import (
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconciler"
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

	// Validate no error metrics are emitted.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating error metrics: %v", err)
	}
}

func TestDeclareImplicitNamespace(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	const implicitNamespace = "shipping"

	err := nt.ValidateNotFound(implicitNamespace, "", &corev1.Namespace{})
	if err != nil {
		// Failed test precondition. We want to ensure we create the Namespace.
		t.Fatal(err)
	}

	// Phase 1: Declare a Role in a Namespace that doesn't exist, and ensure it
	// gets created.
	nt.Root.Add("acme/role.yaml", fake.RoleObject(core.Name("admin"),
		core.Namespace(implicitNamespace)))
	nt.Root.CommitAndPush("add Role in implicit Namespace")
	nt.WaitForRepoSyncs()

	err = nt.Validate(implicitNamespace, "", &corev1.Namespace{})
	if err != nil {
		// No need to continue test since Namespace was never created.
		t.Fatal(err)
	}
	err = nt.Validate("admin", implicitNamespace, &rbacv1.Role{})
	if err != nil {
		t.Error(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("Role"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// Phase 2: Remove the Role, and ensure the implicit Namespace is NOT deleted.
	nt.Root.Remove("acme/role.yaml")
	nt.Root.CommitAndPush("remove Role")
	nt.WaitForRepoSyncs()

	err = nt.Validate(implicitNamespace, "", &corev1.Namespace{})
	if err != nil {
		t.Error(err)
	}
	err = nt.ValidateNotFound("admin", implicitNamespace, &rbacv1.Role{})
	if err != nil {
		t.Error(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 0, metrics.ResourceDeleted("Role"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//if err := nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
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

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.GVKMetric{
				GVK:   "Namespace",
				APIOp: "update",
				ApplyOps: []metrics.Operation{
					{Name: "update", Count: 2},
				},
				Watches: "1",
			})
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
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

	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		// Validate parse error metric is emitted.
		err := nt.ValidateParseErrors(reconciler.RootSyncName, "2006")
		if err != nil {
			t.Errorf("validating parse_errors_total metric: %v", err)
		}
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "sync")
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
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

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 1,
			metrics.ResourceCreated("Namespace"),
			metrics.GVKMetric{
				GVK:      "Namespace",
				APIOp:    "",
				ApplyOps: []metrics.Operation{{Name: "update", Count: 4}},
				Watches:  "1",
			},
			metrics.GVKMetric{
				GVK:      "Namespace",
				APIOp:    "delete",
				ApplyOps: []metrics.Operation{{Name: "delete", Count: 1}},
				Watches:  "1",
			})
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
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

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 0,
			metrics.GVKMetric{
				GVK:      "Namespace",
				APIOp:    "delete",
				ApplyOps: []metrics.Operation{{Name: "delete", Count: 2}},
				Watches:  "0",
			})
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
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
