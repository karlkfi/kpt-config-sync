package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNamespaceRepo_Centralized(t *testing.T) {
	bsNamespace := "bookstore"

	nt := nomostest.New(
		t,
		ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(bsNamespace),
		ntopts.WithCentralizedControl,
	)

	// Validate status condition "Reconciling" and "Stalled "is set to "False"
	// after the reconciler deployment is successfully created.
	// RepoSync status conditions "Reconciling" and "Stalled" are derived from
	// namespace reconciler deployment.
	// Log error if the Reconciling condition does not progress to False before
	// the timeout expires.
	_, err := nomostest.Retry(15*time.Second, func() error {
		return nt.Validate("repo-sync", bsNamespace, &v1alpha1.RepoSync{},
			hasReconcilingStatus(metav1.ConditionFalse), hasStalledStatus(metav1.ConditionFalse))
	})
	if err != nil {
		t.Errorf("RepoSync did not finish reconciling: %v", err)
	}

	repo, exist := nt.NonRootRepos[bsNamespace]
	if !exist {
		t.Fatal("nonexistent repo")
	}

	// Validate service account 'store' not present.
	err = nt.ValidateNotFound("store", bsNamespace, &corev1.ServiceAccount{})
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

	// Validate multi-repo metrics from namespace reconciler.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RepoSyncName(bsNamespace), 1, metrics.ResourceCreated("ServiceAccount"))
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

func hasReconcilingStatus(r metav1.ConditionStatus) nomostest.Predicate {
	return func(o client.Object) error {
		rs := o.(*v1alpha1.RepoSync)
		conditions := rs.Status.Conditions
		for _, condition := range conditions {
			if condition.Type == v1alpha1.RepoSyncReconciling && condition.Status != r {
				return fmt.Errorf("object %q has %q condition status %q; want %q", o.GetName(), condition.Type, string(condition.Status), r)
			}
		}
		return nil
	}
}

func hasStalledStatus(r metav1.ConditionStatus) nomostest.Predicate {
	return func(o client.Object) error {
		rs := o.(*v1alpha1.RepoSync)
		conditions := rs.Status.Conditions
		for _, condition := range conditions {
			if condition.Type == v1alpha1.RepoSyncStalled && condition.Status != r {
				return fmt.Errorf("object %q has %q condition status %q; want %q", o.GetName(), condition.Type, string(condition.Status), r)
			}
		}
		return nil
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

	// Validate multi-repo metrics from namespace reconciler.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RepoSyncName(bsNamespaceRepo), 1, metrics.ResourceCreated("ServiceAccount"))
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

	// Validate multi-repo metrics from root reconciler.
	err := nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 3, metrics.ResourceDeleted("RepoSync"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): unexpected internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}
}

func checkRepoSyncResourcesNotPresent(namespace string, nt *nomostest.NT) {
	_, err := nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(v1alpha1.RepoSyncName, namespace, fake.RepoSyncObject())
	})
	if err != nil {
		nt.T.Errorf("RepoSync present after deletion: %v", err)
	}

	// Verify Namespace Reconciler service no longer present.
	_, err = nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(reconciler.RepoSyncName(namespace), v1.NSConfigManagementSystem, fake.ServiceObject())
	})
	if err != nil {
		nt.T.Errorf("Reconciler service present after deletion: %v", err)
	}

	// Verify Namespace Reconciler deployment no longer present.
	_, err = nomostest.Retry(5*time.Second, func() error {
		return nt.ValidateNotFound(reconciler.RepoSyncName(namespace), v1.NSConfigManagementSystem, fake.DeploymentObject())
	})
	if err != nil {
		nt.T.Errorf("Reconciler deployment present after deletion: %v", err)
	}

	// validate Namespace Reconciler configmaps are no longer present.
	err1 := nt.ValidateNotFound("ns-reconciler-bookstore-git-sync", configsync.ControllerNamespace, fake.ConfigMapObject())
	err2 := nt.ValidateNotFound("ns-reconciler-bookstore-reconciler", configsync.ControllerNamespace, fake.ConfigMapObject())
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

func TestDeleteNamespaceReconcilerDeployment(t *testing.T) {
	bsNamespace := "bookstore"
	nt := nomostest.New(
		t,
		ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(bsNamespace),
		ntopts.WithCentralizedControl,
	)

	// Validate status condition "Reconciling" and Stalled is set to "False" after
	// the reconciler deployment is successfully created.
	// RepoSync status conditions "Reconciling" and "Stalled" are derived from
	// namespace reconciler deployment.
	// Retry before checking for Reconciling and Stalled conditions since the
	// reconcile request is received upon change in the reconciler deployment
	// conditions.
	// Here we are checking for false condition which requires atleast 2 reconcile
	// request to be processed by the controller.
	_, err := nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(v1alpha1.RepoSyncName, bsNamespace, &v1alpha1.RepoSync{},
			hasReconcilingStatus(metav1.ConditionFalse), hasStalledStatus(metav1.ConditionFalse))
	})
	if err != nil {
		t.Errorf("RepoSync did not finish reconciling: %v", err)
	}

	// Delete namespace reconciler deployment in bookstore namespace.
	// The point here is to test that we properly respond to kubectl commands,
	// so this should NOT be replaced with nt.Delete.
	nsReconcilerDeployment := "ns-reconciler-bookstore"
	nt.MustKubectl("delete", "deployment", nsReconcilerDeployment,
		"-n", configsync.ControllerNamespace)

	// Verify that the deployment is re-created after deletion by checking the
	// Reconciling and Stalled condition in RepoSync resource.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(v1alpha1.RepoSyncName, bsNamespace, &v1alpha1.RepoSync{},
			hasReconcilingStatus(metav1.ConditionFalse), hasStalledStatus(metav1.ConditionFalse))
	})
	if err != nil {
		t.Errorf("RepoSync did not finish reconciling: %v", err)
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
