package nomostest

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e"
)

// New establishes a connection to a test cluster and prepares it for testing.
//
// The following are guaranteed to be available when this function returns:
// 1) A connection to the Kubernetes cluster.
// 2) A functioning git server hosted on the cluster.
// 3) TODO(willbeason): A fresh ACM installation.
func New(t *testing.T) *NT {
	t.Helper()

	if !*e2e.E2E {
		// This safeguard prevents test authors from accidentally forgetting to
		// check for the e2e test flag, so `go test ./...` functions as expected
		// without triggering e2e tests.
		t.Fatal("Attempting to createKindCluster cluster for non-e2e test. To fix, copy TestMain() from another e2e test.")
	}

	name := testName(t)
	tmpDir := testDir(t, name)

	// TODO(willbeason): Support connecting to:
	//  1) A user-specified cluster.
	//  2) One of a set of already-set-up clusters?
	cfg := newKind(t, name, tmpDir)
	c := connect(t, cfg)

	nt := &NT{
		Context: context.Background(),
		T:       t,
		Name:    name,
		TmpDir:  tmpDir,
		Client:  c,
	}

	connectToLocalRegistry(nt)
	waitForGit := installGitServer(nt)
	waitForConfigSync := installConfigSync(nt)

	// Wait for git-server to become available before proceeding with the test.
	err := waitForAll(waitForGit, waitForConfigSync)
	if err != nil {
		t.Fatal(err)
	}
	return nt
}

func waitForAll(waits ...func() error) error {
	for _, w := range waits {
		err := w()
		if err != nil {
			return err
		}
	}
	return nil
}

func testDir(t *testing.T, name string) string {
	t.Helper()

	subPath := filepath.Join("nomos-e2e", name+"-")
	err := os.MkdirAll(filepath.Join(os.TempDir(), subPath), os.ModePerm)
	if err != nil {
		t.Fatalf("creating nomos-e2e tmp directory: %v", err)
	}
	tmpDir, err := ioutil.TempDir(os.TempDir(), subPath)
	if err != nil {
		t.Fatalf("creating nomos-e2e tmp test subdirectory: %v", err)
	}
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			// If you're seeing this error, the test does something that prevents
			// cleanup. This is a potential leak for interaction between tests.
			t.Errorf("removing temporary directory %q: %v", tmpDir, err)
		}
	})
	t.Logf("created temporary directory %q", tmpDir)
	return tmpDir
}
