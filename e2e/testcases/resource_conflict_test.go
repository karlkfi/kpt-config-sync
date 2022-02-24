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
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/status"
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
	nt := nomostest.New(t, ntopts.NamespaceRepo("shipping", configsync.RepoSyncName), ntopts.SkipMonoRepo)

	// Add a Role to root.
	nt.RootRepos[configsync.RootSyncName].Add("acme/namespaces/shipping/pod-role.yaml", rootPodRole())
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add pod viewer role")
	nt.WaitForRepoSyncs()

	// Validate multi-repo metrics from root reconciler.
	err := nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		var err error
		// TODO(b/193186006): Remove the psp related change when Kubernetes 1.25 is
		// available on GKE.
		if strings.Contains(os.Getenv("GCP_CLUSTER"), "psp") {
			err = nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName, 7, metrics.ResourceCreated("Role"))
		} else {
			err = nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName, 6, metrics.ResourceCreated("Role"))
		}
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	// Declare a conflicting Role in the shipping Namespace repo.
	nt.NonRootRepos[nomostest.RepoSyncNN("shipping", configsync.RepoSyncName)].Add("acme/namespaces/shipping/pod-role.yaml", namespacePodRole())
	nt.NonRootRepos[nomostest.RepoSyncNN("shipping", configsync.RepoSyncName)].CommitAndPush("add conflicting pod owner role")

	// The RootSync should report no problems - it has no way to detect there is
	// a problem.
	nt.WaitForRepoSyncs(nomostest.RootSyncOnly())

	// The shipping RepoSync reports a problem since it can't sync the declaration.
	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate(configsync.RepoSyncName, "shipping", &v1beta1.RepoSync{},
			repoSyncHasErrors(status.ManagementConflictErrorCode))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate reconciler error metric is emitted from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateReconcilerErrors(reconciler.NsReconcilerName("shipping", configsync.RepoSyncName), "sync")
	})
	if err != nil {
		nt.T.Errorf("validating reconciler_errors metric: %v", err)
	}

	// Ensure the Role matches the one in the Root repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{}, roleHasRules(rootPodRole().Rules))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Remove the declaration from the Root repo.
	nt.RootRepos[configsync.RootSyncName].Remove("acme/namespaces/shipping/pod-role.yaml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("remove conflicting pod role from Root")
	nt.WaitForRepoSyncs()

	// Ensure the Role is updated to the one in the Namespace repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{},
		roleHasRules(namespacePodRole().Rules))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate multi-repo metrics from root reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		var err error
		// TODO(b/193186006): Remove the psp related change when Kubernetes 1.25 is
		// available on GKE.
		if strings.Contains(os.Getenv("GCP_CLUSTER"), "psp") {
			err = nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName, 6, metrics.ResourceDeleted("Role"))
		} else {
			err = nt.ValidateMultiRepoMetrics(nomostest.DefaultRootReconcilerName, 5, metrics.ResourceDeleted("Role"))
		}
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}
}

func TestConflictingDefinitions_NamespaceToRoot(t *testing.T) {
	nt := nomostest.New(t, ntopts.NamespaceRepo("shipping", configsync.RepoSyncName), ntopts.SkipMonoRepo)

	// Add a Role to Namespace.
	nt.NonRootRepos[nomostest.RepoSyncNN("shipping", configsync.RepoSyncName)].Add("acme/namespaces/shipping/pod-role.yaml", namespacePodRole())
	nt.NonRootRepos[nomostest.RepoSyncNN("shipping", configsync.RepoSyncName)].CommitAndPush("declare Role")
	nt.WaitForRepoSyncs()

	err := nt.Validate("pods", "shipping", &rbacv1.Role{},
		roleHasRules(namespacePodRole().Rules))
	if err != nil {
		nt.T.Fatal(err)
	}

	nsReconcilerName := reconciler.NsReconcilerName("shipping", configsync.RepoSyncName)
	// Validate multi-repo metrics from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nsReconcilerName, 1, metrics.ResourceCreated("Role"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	nt.RootRepos[configsync.RootSyncName].Add("acme/namespaces/shipping/pod-role.yaml", rootPodRole())
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add conflicting pod role to Root")

	nt.WaitForRepoSyncs(nomostest.RootSyncOnly())
	// The shipping RepoSync reports a problem since it can't sync the declaration.
	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate(configsync.RepoSyncName, "shipping", &v1beta1.RepoSync{},
			repoSyncHasErrors(status.ManagementConflictErrorCode))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate reconciler error metric is emitted from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateReconcilerErrors(nsReconcilerName, "sync")
	})
	if err != nil {
		nt.T.Errorf("validating reconciler_errors metric: %v", err)
	}

	// Ensure the Role matches the one in the Root repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{}, roleHasRules(rootPodRole().Rules))
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.NonRootRepos[nomostest.RepoSyncNN("shipping", configsync.RepoSyncName)].Remove("acme/namespaces/shipping/pod-role.yaml")
	nt.NonRootRepos[nomostest.RepoSyncNN("shipping", configsync.RepoSyncName)].CommitAndPush("remove conflicting pod role from Namespace repo")
	nt.WaitForRepoSyncs()

	// Ensure the Role still matches the one in the Root repo.
	err = nt.Validate("pods", "shipping", &rbacv1.Role{},
		roleHasRules(rootPodRole().Rules))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Validate multi-repo metrics from namespace reconciler.
	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(nsReconcilerName, 0)
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
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

	var wantErrs []v1beta1.ConfigSyncError
	for _, code := range wantCodes {
		wantErrs = append(wantErrs, v1beta1.ConfigSyncError{Code: code})
	}

	return func(o client.Object) error {
		rs, isRepoSync := o.(*v1beta1.RepoSync)
		if !isRepoSync {
			return nomostest.WrongTypeErr(o, &v1beta1.RepoSync{})
		}

		gotErrs := rs.Status.Sync.Errors
		sort.Slice(gotErrs, func(i, j int) bool {
			return gotErrs[i].Code < gotErrs[j].Code
		})

		if diff := cmp.Diff(wantErrs, gotErrs,
			cmpopts.IgnoreFields(v1beta1.ConfigSyncError{},
				"ErrorMessage", "Resources")); diff != "" {
			return errors.New(diff)
		}
		return nil
	}
}
