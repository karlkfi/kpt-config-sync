package nomostest

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/testing/fake"
)

// fileMode is the file mode to use for all operations.
//
// Go is inconsistent about which user/group it runs commands under, so
// anything less will either:
// 1) Make git operations not work as expected, or
// 2) Cause ssh-keygen to fail.
const fileMode = os.ModePerm

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
	// You can't add Secrets to Namespaces that don't exist, so create them now.
	err := nt.Create(fake.NamespaceObject(configmanagement.ControllerNamespace))
	if err != nil {
		nt.T.Fatal(err)
	}
	err = nt.Create(gitNamespace())
	if err != nil {
		nt.T.Fatal(err)
	}
	// Pods don't always restart if the secrets don't exist, so we have to
	// create the Namespaces + Secrets before anything else.
	generateSSHKeys(nt, filepath.Join(tmpDir, kubeconfig))

	waitForGit := installGitServer(nt)
	waitForConfigSync := installConfigSync(nt)

	err = waitForGit()
	if err != nil {
		t.Fatalf("waiting for git-server Deployment to become available: %v", err)
	}
	// The git-server reports itself to be ready, so we don't have to wait on
	// anything.
	port := portForwardGitServer(nt)
	nt.Git = NewRepository(t, "sot.git", nt.TmpDir, port)

	err = waitForConfigSync()
	if err != nil {
		t.Fatalf("waiting for ConfigSync Deployments to become available: %v", err)
	}

	return nt
}

func testDir(t *testing.T, name string) string {
	t.Helper()

	subPath := filepath.Join("nomos-e2e", name+"-")
	err := os.MkdirAll(filepath.Join(os.TempDir(), subPath), fileMode)
	if err != nil {
		t.Fatalf("creating nomos-e2e tmp directory: %v", err)
	}
	tmpDir, err := ioutil.TempDir(os.TempDir(), subPath)
	if err != nil {
		t.Fatalf("creating nomos-e2e tmp test subdirectory: %v", err)
	}
	t.Cleanup(func() {
		if t.Failed() && *e2e.Debug {
			t.Errorf("temporary directory: %s", tmpDir)
			return
		}
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
