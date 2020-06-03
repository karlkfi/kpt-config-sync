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

// TestDeclareNamespace (once complete), will run a test that ensures ACM syncs
// Namespaces to clusters.
func TestDeclareNamespace(t *testing.T) {
	// TODO(b/1041731): Build Nomos Image + Generate Manifests.
	//  Set up Git Server.
	//  Apply Nomos manifests.
	//  Wait for ConfigManagement CRD to be available + apply CM.
	//  Set up Git repo initial state.
	c := nomostest.New(t)

	err := c.ValidateNotFound("foo", "", &corev1.Namespace{})
	if err != nil {
		t.Error(err)
	}

	// TODO(b/1041731): Declare Namespace "foo" in repo
	//  Wait for ACM to report it is synced.
	// The below will be done by ACM in the final test.
	err = c.Create(fake.NamespaceObject("foo"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		err := c.Delete(fake.NamespaceObject("foo"))
		if err != nil {
			t.Error(err)
		}
	})

	// Test that the Namespace "foo" exists.
	err = c.Validate("foo", "", &corev1.Namespace{})
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
