package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/cli-utils/pkg/common"
)

var preventDeletion = core.Annotation(common.LifecycleDeleteAnnotation, common.PreventDeletion)

func TestPreventDeletionNamespace(t *testing.T) {
	nt := nomostest.New(t)

	// Ensure the Namespace doesn't already exist.
	err := nt.ValidateNotFound("shipping", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}

	role := fake.RoleObject(core.Name("shipping-admin"))
	role.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}

	// Declare the Namespace with the lifecycle annotation, and ensure it is created.
	nt.Root.Add("acme/namespaces/shipping/ns.yaml",
		fake.NamespaceObject("shipping", preventDeletion))
	nt.Root.Add("acme/namespaces/shipping/role.yaml", role)
	nt.Root.CommitAndPush("declare Namespace with prevent deletion lifecycle annotation")
	nt.WaitForRepoSyncs()

	err = nt.Validate("shipping", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
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

	// Delete the declaration and ensure the Namespace isn't deleted.
	nt.Root.Remove("acme/namespaces/shipping/ns.yaml")
	nt.Root.Remove("acme/namespaces/shipping/role.yaml")
	nt.Root.CommitAndPush("remove Namespace shipping declaration")
	nt.WaitForRepoSyncs()

	// Ensure we kept the undeclared Namespace that had the "deletion: prevent" annotation.
	err = nt.Validate("shipping", "", &corev1.Namespace{},
		nomostest.NotPendingDeletion)
	if err != nil {
		t.Fatal(err)
	}
	// Ensure we deleted the undeclared Role that doesn't have the annotation.
	err = nt.ValidateNotFound("shipping-admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 0,
			metrics.ResourceDeleted("Role"),
			metrics.GVKMetric{
				GVK:      "Namespace",
				APIOp:    "update",
				ApplyOps: []metrics.Operation{{Name: "update", Count: 2}},
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

func TestPreventDeletionRole(t *testing.T) {
	nt := nomostest.New(t)

	// Ensure the Namespace doesn't already exist.
	err := nt.ValidateNotFound("shipping-admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
	}

	// Declare the Role with the lifecycle annotation, and ensure it is created.
	role := fake.RoleObject(core.Name("shipping-admin"), preventDeletion)
	role.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}
	nt.Root.Add("acme/namespaces/shipping/ns.yaml", fake.NamespaceObject("shipping"))
	nt.Root.Add("acme/namespaces/shipping/role.yaml", role)
	nt.Root.CommitAndPush("declare Role with prevent deletion lifecycle annotation")
	nt.WaitForRepoSyncs()

	err = nt.Validate("shipping-admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
	}

	// Delete the declaration and ensure the Namespace isn't deleted.
	nt.Root.Remove("acme/namespaces/shipping/role.yaml")
	nt.Root.CommitAndPush("remove Role declaration")
	nt.WaitForRepoSyncs()

	err = nt.Validate("shipping-admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
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

func TestPreventDeletionClusterRole(t *testing.T) {
	nt := nomostest.New(t)

	// Ensure the ClusterRole doesn't already exist.
	err := nt.ValidateNotFound("test-admin", "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
	}

	// Declare the ClusterRole with the lifecycle annotation, and ensure it is created.
	clusterRole := fake.ClusterRoleObject(core.Name("test-admin"), preventDeletion)
	clusterRole.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}
	nt.Root.Add("acme/cluster/cr.yaml", clusterRole)
	nt.Root.CommitAndPush("declare ClusterRole with prevent deletion lifecycle annotation")
	nt.WaitForRepoSyncs()

	err = nt.Validate("test-admin", "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
	}

	// Delete the declaration and ensure the ClusterRole isn't deleted.
	nt.Root.Remove("acme/cluster/cr.yaml")
	nt.Root.CommitAndPush("remove ClusterRole bar declaration")
	nt.WaitForRepoSyncs()

	err = nt.Validate("test-admin", "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
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

func TestPreventDeletionImplicitNamespace(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured, ntopts.SkipMultiRepo)

	const implicitNamespace = "delivery"

	role := fake.RoleObject(core.Name("configmap-getter"), core.Namespace(implicitNamespace))
	role.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}
	nt.Root.Add("acme/role.yaml", role)
	nt.Root.CommitAndPush("Declare configmap-getter Role")
	nt.WaitForRepoSyncs()

	err := nt.Validate(implicitNamespace, "", &corev1.Namespace{},
		nomostest.HasAnnotation(common.LifecycleDeleteAnnotation, common.PreventDeletion))
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("configmap-getter", implicitNamespace, &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
	}

	nt.Root.Remove("acme/role.yaml")
	nt.Root.CommitAndPush("Remove configmap-getter Role")
	nt.WaitForRepoSyncs()

	// Ensure the Namespace wasn't deleted.
	err = nt.Validate(implicitNamespace, "", &corev1.Namespace{},
		nomostest.NotPendingDeletion)
	if err != nil {
		t.Fatal(err)
	}
}
