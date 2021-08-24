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

	nt.T.Cleanup(removeKubeVirtCR(nt, "acme/namespaces/kubevirt/kubevirt-cr.yaml"))

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

	nt.T.Cleanup(removeKubeVirtCR(nt, "acme/kubevirt/kubevirt_kubevirt.yaml"))

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

func removeKubeVirtCR(nt *nomostest.NT, path string) func() {
	return func() {
		nt.Root.Remove(path)
		nt.Root.CommitAndPush("Remove kubevirt custom resource")
		nt.WaitForRepoSyncs(nomostest.WithTimeout(4 * time.Minute))
	}
}
