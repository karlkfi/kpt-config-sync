package nomostest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e"
)

// New establishes a connection to a test cluster.
func New(t *testing.T) *NT {
	t.Helper()

	if !*e2e.E2E {
		// This safeguard prevents test authors from accidentally forgetting to
		// check for the e2e test flag, so `go test ./...` functions as expected
		// without triggering e2e tests.
		t.Fatal("Attempting to create cluster for non-e2e test. To fix, copy TestMain() from another e2e test.")
	}

	name := testName(t)
	tmpDir := testDir(t, name)

	// TODO(willbeason): Support connecting to:
	//  1) A user-specified cluster.
	//  2) One of a set of already-set-up clusters?
	cfg := newKind(t, name, tmpDir)
	c := connect(t, cfg)

	return &NT{
		Name:   name,
		TmpDir: tmpDir,
		Client: c,
	}
}

func testDir(t *testing.T, name string) string {
	subPath := filepath.Join("nomos-e2e", name+"-")
	tmpDir, err := ioutil.TempDir(os.TempDir(), subPath)
	if err != nil {
		t.Fatal(err)
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
