package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
)

func TestApplyScopedResourcesHierarchicalMode(t *testing.T) {
	nt := nomostest.New(t)

	nt.Root.Remove("acme/namespaces")
	nt.Root.Copy("../../examples/kubevirt/.", "acme")
	nt.Root.CommitAndPush("Add kubevirt configs")
	nt.WaitForRepoSyncs()

	nt.T.Cleanup(func() {
		// Avoids KNV2010 error since the bookstore namespace contains a VM custom resource
		// KNV2010: unable to apply resource: the server could not find the requested resource (patch virtualmachines.kubevirt.io testvm)
		// Error occurs semi-consistently (~50% of the time) with the CI mono-repo kind tests
		nt.Root.Remove("acme/namespaces/bookstore1")
		nt.Root.CommitAndPush("Remove bookstore1 namespace")
		nt.WaitForRepoSyncs()

		// kubevirt must be removed separately to allow the custom resource to be deleted
		nt.Root.Remove("acme/namespaces/kubevirt/kubevirt-cr.yaml")
		nt.Root.CommitAndPush("Remove kubevirt custom resource")
		nt.WaitForRepoSyncs()

		// Prevents the kubevirt custom resource from being stuck in 'Deleting' phase which can
		// occur if the cluster scoped resources are removed prior to the custom resource being
		// deleted. This cannot be combined with the same commit as removing the custom resource
		// since the custom resource has a finalizer that depends on the operator existing.
		nt.Root.Remove("acme/namespaces/kubevirt/kubevirt-operator.yaml")
		nt.Root.CommitAndPush("Remove kubevirt operator")
		nt.WaitForRepoSyncs()

		// Avoids KNV2006 since the repo contains a number of cluster scoped resources
		// https://cloud.google.com/anthos-config-management/docs/reference/errors#knv2006
		nt.Root.Remove("acme/cluster/kubevirt-operator-cluster-role.yaml")
		nt.Root.Remove("acme/cluster/kubevirt.io:operator-clusterrole.yaml")
		nt.Root.Remove("acme/cluster/kubevirt-cluster-critical.yaml")
		nt.Root.CommitAndPush("Remove cluster roles and priority class")
		nt.WaitForRepoSyncs()
	})

	err := nomostest.WaitForCRDs(nt, []string{"virtualmachines.kubevirt.io"})
	if err != nil {
		nt.T.Fatal(err)
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		_, err := nt.Kubectl("get", "vm", "testvm", "-n", "bookstore1")
		return err
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func TestApplyScopedResourcesUnstructuredMode(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	nt.Root.Copy("../../examples/kubevirt-compiled/.", "acme")
	nt.Root.CommitAndPush("Add kubevirt configs")
	nt.WaitForRepoSyncs()

	nt.T.Cleanup(func() {
		// Avoids KNV2010 error since the bookstore namespace contains a VM custom resource
		// KNV2010: unable to apply resource: the server could not find the requested resource (patch virtualmachines.kubevirt.io testvm)
		// Error occurs semi-consistently (~50% of the time) with the CI mono-repo kind tests
		nt.Root.Remove("acme/namespace_bookstore1.yaml")
		nt.Root.Remove("acme/bookstore1")
		nt.Root.CommitAndPush("Remove bookstore1 namespace")
		nt.WaitForRepoSyncs()

		// kubevirt must be removed separately to allow the custom resource to be deleted
		nt.Root.Remove("acme/kubevirt/kubevirt_kubevirt.yaml")
		nt.Root.CommitAndPush("Remove kubevirt custom resource")
		nt.WaitForRepoSyncs()

		// Prevents the kubevirt custom resource from being stuck in 'Deleting' phase which can
		// occur if the cluster scoped resources are removed prior to the custom resource being
		// deleted. This cannot be combined with the same commit as removing the custom resource
		// since the custom resource has a finalizer that depends on the operator existing.
		nt.Root.Remove("acme/kubevirt/deployment_virt-operator.yaml")
		nt.Root.Remove("acme/kubevirt/role_kubevirt-operator.yaml")
		nt.Root.Remove("acme/kubevirt/rolebinding_kubevirt-operator-rolebinding.yaml")
		nt.Root.Remove("acme/kubevirt/serviceaccount_kubevirt-operator.yaml")
		nt.Root.CommitAndPush("Remove kubevirt operator")
		nt.WaitForRepoSyncs()

		// Avoids KNV2006 since the repo contains a number of cluster scoped resources
		// https://cloud.google.com/anthos-config-management/docs/reference/errors#knv2006
		nt.Root.Remove("acme/clusterrole_kubevirt-operator.yaml")
		nt.Root.Remove("acme/clusterrole_kubevirt.io:operator.yaml")
		nt.Root.Remove("acme/clusterrolebinding_kubevirt-operator.yaml")
		nt.Root.Remove("acme/priorityclass_kubevirt-cluster-critical.yaml")
		nt.Root.CommitAndPush("Remove cluster roles and priority class")
		nt.WaitForRepoSyncs()
	})

	err := nomostest.WaitForCRDs(nt, []string{"virtualmachines.kubevirt.io"})
	if err != nil {
		nt.T.Fatal(err)
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		_, err := nt.Kubectl("get", "vm", "testvm", "-n", "bookstore1")
		return err
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
