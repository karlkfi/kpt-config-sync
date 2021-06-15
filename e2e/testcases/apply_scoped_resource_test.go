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

	err := nomostest.WaitForCRDs(nt, []string{"virtualmachines.kubevirt.io"})
	if err != nil {
		t.Fatal(err)
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

	err := nomostest.WaitForCRDs(nt, []string{"virtualmachines.kubevirt.io"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		_, err := nt.Kubectl("get", "vm", "testvm", "-n", "bookstore1")
		return err
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
