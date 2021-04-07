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
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func rootPodRole() *rbacv1.Role {
	result := fake.RoleObject(
		core.Name("pods"),
		core.Namespace("shipping"),
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
		core.Namespace("shipping"),
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

func TestConflictingDefinitions_RootToNamespace(t *testing.T) {
	nt := nomostest.New(t, ntopts.NamespaceRepo("shipping"), ntopts.SkipMonoRepo)

	// Add a Role to root.
	nt.Root.Add("acme/namespaces/shipping/pod-role.yaml", rootPodRole())
	nt.Root.CommitAndPush("add pod viewer role")
	nt.WaitForRepoSyncs()

	// Validate multi-repo metrics from root reconciler.
	err := nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 5, metrics.ResourceCreated("Role"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// Declare a conflicting Role in the shipping Namespace repo.
	nt.NonRootRepos["shipping"].Add("acme/namespaces/shipping/pod-role.yaml", namespacePodRole())
	nt.NonRootRepos["shipping"].CommitAndPush("add conflicting pod owner role")

	// The RootSync should report no problems - it has no way to detect there is
	// a problem.
	nt.WaitForRootSync(kinds.RootSync(),
		"root-sync", configmanagement.ControllerNamespace, nomostest.RootSyncHasStatusSyncCommit)

	// The shipping RepoSync reports a problem since it can't sync the declaration.
	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate(v1alpha1.RepoSyncName, "shipping", &v1alpha1.RepoSync{},
			repoSyncHasErrors(applier.ManagementConflictErrorCode))
	})
	if err != nil {
		t.Fatal(err)
	}

	// Validate reconciler error metric is emitted from namespace reconciler.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateReconcilerErrors(reconciler.RepoSyncName("shipping"), "sync")
	})
	if err != nil {
		t.Errorf("validating reconciler_errors metric: %v", err)
	}

	// Ensure the Role matches the one in the Root repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{}, roleHasRules(rootPodRole().Rules))
	if err != nil {
		t.Fatal(err)
	}

	// Remove the declaration from the Root repo.
	nt.Root.Remove("acme/namespaces/shipping/pod-role.yaml")
	nt.Root.CommitAndPush("remove conflicting pod role from Root")
	nt.WaitForRepoSyncs()

	// Ensure the Role is updated to the one in the Namespace repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{},
		roleHasRules(namespacePodRole().Rules))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics from root reconciler.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 4, metrics.ResourceDeleted("Role"))
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

func TestConflictingDefinitions_NamespaceToRoot(t *testing.T) {
	nt := nomostest.New(t, ntopts.NamespaceRepo("shipping"), ntopts.SkipMonoRepo)

	// Add a Role to Namespace.
	nt.NonRootRepos["shipping"].Add("acme/namespaces/shipping/pod-role.yaml", namespacePodRole())
	nt.NonRootRepos["shipping"].CommitAndPush("declare Role")
	nt.WaitForRepoSyncs()

	err := nt.Validate("pods", "shipping", &rbacv1.Role{},
		roleHasRules(namespacePodRole().Rules))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics from namespace reconciler.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RepoSyncName("shipping"), 1, metrics.ResourceCreated("Role"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	nt.Root.Add("acme/namespaces/shipping/pod-role.yaml", rootPodRole())
	nt.Root.CommitAndPush("add conflicting pod role to Root")

	nt.WaitForRootSync(kinds.RootSync(),
		"root-sync", configmanagement.ControllerNamespace, nomostest.RootSyncHasStatusSyncCommit)
	// The shipping RepoSync reports a problem since it can't sync the declaration.
	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate(v1alpha1.RepoSyncName, "shipping", &v1alpha1.RepoSync{},
			repoSyncHasErrors(applier.ManagementConflictErrorCode))
	})
	if err != nil {
		t.Fatal(err)
	}

	// Validate reconciler error metric is emitted from namespace reconciler.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateReconcilerErrors(reconciler.RepoSyncName("shipping"), "sync")
	})
	if err != nil {
		t.Errorf("validating reconciler_errors metric: %v", err)
	}

	// Ensure the Role matches the one in the Root repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{}, roleHasRules(rootPodRole().Rules))
	if err != nil {
		t.Fatal(err)
	}

	nt.NonRootRepos["shipping"].Remove("acme/namespaces/shipping/pod-role.yaml")
	nt.NonRootRepos["shipping"].CommitAndPush("remove conflicting pod role from Namespace repo")
	nt.WaitForRepoSyncs()

	// Ensure the Role still matches the one in the Root repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{},
		roleHasRules(rootPodRole().Rules))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics from namespace reconciler.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RepoSyncName("shipping"), 0)
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}
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

func repoSyncHasErrors(wantCodes ...string) nomostest.Predicate {
	sort.Strings(wantCodes)

	var wantErrs []v1alpha1.ConfigSyncError
	for _, code := range wantCodes {
		wantErrs = append(wantErrs, v1alpha1.ConfigSyncError{Code: code})
	}

	return func(o client.Object) error {
		rs, isRepoSync := o.(*v1alpha1.RepoSync)
		if !isRepoSync {
			return nomostest.WrongTypeErr(o, &v1alpha1.RepoSync{})
		}

		gotErrs := rs.Status.Sync.Errors
		sort.Slice(gotErrs, func(i, j int) bool {
			return gotErrs[i].Code < gotErrs[j].Code
		})

		if diff := cmp.Diff(wantErrs, gotErrs,
			cmpopts.IgnoreFields(v1alpha1.ConfigSyncError{},
				"ErrorMessage", "Resources")); diff != "" {
			return errors.New(diff)
		}
		return nil
	}
}
