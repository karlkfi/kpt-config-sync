package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/lifecycle"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var preventDeletion = core.Annotation(lifecycle.Deletion, lifecycle.PreventDeletion)

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
	nt.Repository.Add("acme/namespaces/shipping/ns.yaml",
		fake.NamespaceObject("shipping", preventDeletion))
	nt.Repository.Add("acme/namespaces/shipping/role.yaml", role)
	nt.Repository.CommitAndPush("declare Namespace with prevent deletion lifecycle annotation")
	nt.WaitForRepoSync()

	err = nt.Validate("shipping", "", &corev1.Namespace{})
	if err != nil {
		t.Fatal(err)
	}

	// Delete the declaration and ensure the Namespace isn't deleted.
	nt.Repository.Remove("acme/namespaces/shipping/ns.yaml")
	nt.Repository.Remove("acme/namespaces/shipping/role.yaml")
	nt.Repository.CommitAndPush("remove Namespace shipping declaration")
	nt.WaitForRepoSync()

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
	nt.Repository.Add("acme/namespaces/shipping/ns.yaml", fake.NamespaceObject("shipping"))
	nt.Repository.Add("acme/namespaces/shipping/role.yaml", role)
	nt.Repository.CommitAndPush("declare Role with prevent deletion lifecycle annotation")
	nt.WaitForRepoSync()

	err = nt.Validate("shipping-admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
	}

	// Delete the declaration and ensure the Namespace isn't deleted.
	nt.Repository.Remove("acme/namespaces/shipping/role.yaml")
	nt.Repository.CommitAndPush("remove Role declaration")
	nt.WaitForRepoSync()

	err = nt.Validate("shipping-admin", "shipping", &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
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
	nt.Repository.Add("acme/cluster/cr.yaml", clusterRole)
	nt.Repository.CommitAndPush("declare ClusterRole with prevent deletion lifecycle annotation")
	nt.WaitForRepoSync()

	err = nt.Validate("cluster-admin", "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
	}

	// Delete the declaration and ensure the ClusterRole isn't deleted.
	nt.Repository.Remove("acme/cluster/cr.yaml")
	nt.Repository.CommitAndPush("remove ClusterRole bar declaration")
	nt.WaitForRepoSync()

	err = nt.Validate("test-admin", "", &rbacv1.ClusterRole{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreventDeletionImplicitNamespace(t *testing.T) {
	nt := nomostest.NewWithOptions(t, ntopts.New{
		Nomos: ntopts.Nomos{SourceFormat: filesystem.SourceFormatUnstructured},
	})

	role := fake.RoleObject(core.Name("configmap-getter"), core.Namespace("delivery"))
	role.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"configmaps"},
		Verbs:     []string{"get"},
	}}
	nt.Repository.Add("acme/role.yaml", role)
	nt.Repository.CommitAndPush("Declare configmap-getter Role")
	nt.WaitForRepoSync()

	err := nt.Validate("delivery", "", &corev1.Namespace{},
		nomostest.HasAnnotation(lifecycle.Deletion, lifecycle.PreventDeletion))
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("configmap-getter", "delivery", &rbacv1.Role{})
	if err != nil {
		t.Fatal(err)
	}

	nt.Repository.Remove("acme/role.yaml")
	nt.Repository.CommitAndPush("Remove configmap-getter Role")
	nt.WaitForRepoSync()

	// Ensure the Namespace wasn't deleted.
	err = nt.Validate("delivery", "", &corev1.Namespace{},
		nomostest.NotPendingDeletion)
	if err != nil {
		t.Fatal(err)
	}
}
