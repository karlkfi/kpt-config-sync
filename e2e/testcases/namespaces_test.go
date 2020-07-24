package e2e

import (
	"flag"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
)

// TestDeclareNamespace runs a test that ensures ACM syncs Namespaces to clusters.
func TestDeclareNamespace(t *testing.T) {
	t.Parallel()
	nt := nomostest.New(t)

	err := nt.ValidateNotFound("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Error(err)
	}

	nt.Repository.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Repository.CommitAndPush("add Namespace")
	nt.WaitForRepoSync()

	// Test that the Namespace "foo" exists.
	err = nt.Validate("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Error(err)
	}
}

func TestMain(m *testing.M) {
	// This TestMain function is required in every e2e test case file.
	flag.Parse()

	if !*e2e.E2E {
		return
	}
	rand.Seed(time.Now().UnixNano())

	os.Exit(m.Run())
}
