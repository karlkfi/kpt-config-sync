package nomostest

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/testing/fake"
)

// fileMode is the file mode to use for all operations.
//
// Go is inconsistent about which user/group it runs commands under, so
// anything less will either:
// 1) Make git operations not work as expected, or
// 2) Cause ssh-keygen to fail.
const fileMode = os.ModePerm

// nomosE2E is the subdirectory inside the filesystem's temporary directory in
// which we write test data.
const nomosE2E = "nomos-e2e"

// New establishes a connection to a test cluster and prepares it for testing.
//
// Use NewWithOptions for customization.
func New(t *testing.T) *NT {
	return NewWithOptions(t, ntopts.New{})
}

// NewWithOptions establishes a connection to a test cluster based on the passed
//
// options.
//
// Marks the test as parallel. For now we have no tests which *can't* be made
// parallel; if we need that in the future we can make a version of this
// function that doesn't do this. As below keeps us from forgetting to mark
// tests as parallel, and unnecessarily waiting.
//
// The following are guaranteed to be available when this function returns:
// 1) A connection to the Kubernetes cluster.
// 2) A functioning git server hosted on the cluster.
// 3) A fresh ACM installation.
func NewWithOptions(t *testing.T, opts ntopts.New) *NT {
	t.Parallel()
	t.Helper()

	if !*e2e.E2E {
		// This safeguard prevents test authors from accidentally forgetting to
		// check for the e2e test flag, so `go test ./...` functions as expected
		// without triggering e2e tests.
		t.Fatal("Attempting to createKindCluster cluster for non-e2e test. To fix, copy TestMain() from another e2e test.")
	}

	// Set default options.
	if opts.Name == "" {
		opts.Name = testClusterName(t)
	}
	if opts.TmpDir == "" {
		opts.TmpDir = testDir(t)
	}
	if opts.KindCluster.Version == "" {
		opts.KindCluster.Version = ntopts.AsKindVersion(t, *e2e.KubernetesVersion)
	}
	if opts.Nomos.SourceFormat == "" {
		opts.Nomos.SourceFormat = filesystem.SourceFormatHierarchy
	}

	// TODO(willbeason): Support connecting to:
	//  1) A user-specified cluster.
	//  2) One of a set of already-set-up clusters?
	// We have to update the name since newKind may choose a new name for the
	// cluster if the name is too long.
	cfg, kubeconfigPath := newKind(t, opts.Name, opts.TmpDir, opts.KindCluster)
	c := connect(t, cfg)

	nt := &NT{
		Context:        context.Background(),
		T:              t,
		ClusterName:    opts.Name,
		TmpDir:         opts.TmpDir,
		Config:         cfg,
		Client:         c,
		kubeconfigPath: kubeconfigPath,
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
	nt.gitPrivateKeyPath = generateSSHKeys(nt, filepath.Join(opts.TmpDir, kubeconfig))

	waitForGit := installGitServer(nt)
	waitForConfigSync := installConfigSync(nt, opts.Nomos)

	err = waitForGit()
	if err != nil {
		t.Fatalf("waiting for git-server Deployment to become available: %v", err)
	}
	// The git-server reports itself to be ready, so we don't have to wait on
	// anything.
	nt.gitRepoPort = portForwardGitServer(nt)
	nt.Repository = NewRepository(nt, "sot.git", nt.TmpDir, nt.gitRepoPort, opts.SourceFormat)

	err = waitForConfigSync()
	if err != nil {
		t.Fatalf("waiting for ConfigSync Deployments to become available: %v", err)
	}
	// The Repo type wasn't available when the cluster was initially created.
	// Create a new Client, since it'll automatically be configured to understand
	// the Repo type as ConfigSync is now installed.
	nt.RenewClient()
	nt.WaitForRepoSync()

	return nt
}

func testDir(t *testing.T) string {
	t.Helper()

	name := testDirName(t)
	err := os.MkdirAll(filepath.Join(os.TempDir(), nomosE2E), fileMode)
	if err != nil {
		t.Fatalf("creating nomos-e2e tmp directory: %v", err)
	}
	tmpDir, err := ioutil.TempDir(filepath.Join(os.TempDir(), nomosE2E), name)
	if err != nil {
		t.Fatalf("creating nomos-e2e tmp test subdirectory %s: %v", tmpDir, err)
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
