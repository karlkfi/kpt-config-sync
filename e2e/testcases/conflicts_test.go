package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

// TestConflictingKubectlApply runs a test that ensures ACM merges changes from a separate kubectl apply.
func TestConflictingKubectlApply(t *testing.T) {
	nt := nomostest.New(t)

	err := nt.ValidateNotFound("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Error(err)
	}

	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo", core.Label("hello", "world")))
	nt.Root.CommitAndPush("add hello:world Namespace")
	nt.WaitForRepoSyncs()

	// Test that the Namespace "foo" exists with the expected label.
	err = nt.Validate("foo", "", &corev1.Namespace{}, nomostest.HasLabel("hello", "world"))
	if err != nil {
		t.Error(err)
	}

	err = ioutil.WriteFile(filepath.Join(nt.TmpDir, "conflict.yaml"), []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: foo
  labels:
    goodnight: "moon"
`), os.ModePerm)
	if err != nil {
		t.Fatalf("failed to write temporary yaml file: %v", err)
	}

	// Add a new label via kubectl.
	nt.MustKubectl("apply", "-f", filepath.Join(nt.TmpDir, "conflict.yaml"))

	_, err = nomostest.Retry(time.Second*5, func() error {
		// Test that the Namespace "foo" exists with the manually added label.
		err = nt.Validate("foo", "", &corev1.Namespace{}, nomostest.HasLabel("goodnight", "moon"))
		if err != nil {
			return err
		}

		// Test that the Namespace "foo" still has the label from Git.
		err = nt.Validate("foo", "", &corev1.Namespace{}, nomostest.HasLabel("hello", "world"))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
