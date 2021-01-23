package nomostest

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest/docker"
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
// Use newWithOptions for customization.
func New(t *testing.T, ntOptions ...ntopts.Opt) *NT {
	t.Helper()
	t.Parallel()

	// TODO: we should probably put ntopts.New members inside of NT use the go-convention of mutating NT with option functions.
	optsStruct := ntopts.New{
		Name:   testClusterName(t),
		TmpDir: TestDir(t),
		Nomos: ntopts.Nomos{
			SourceFormat: filesystem.SourceFormatHierarchy,
			MultiRepo:    *e2e.MultiRepo,
		},
		MultiRepo: ntopts.MultiRepo{
			NamespaceRepos: make(map[string]struct{}),
		},
	}

	for _, opt := range ntOptions {
		opt(&optsStruct)
	}

	if optsStruct.RESTConfig == nil {
		ntopts.Kind(t, *e2e.KubernetesVersion)(&optsStruct)
	}
	optsStruct.RESTConfig.QPS = 50
	optsStruct.RESTConfig.Burst = 75

	return newWithOptions(t, optsStruct)
}

// newWithOptions establishes a connection to a test cluster based on the passed
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
func newWithOptions(t *testing.T, opts ntopts.New) *NT {
	t.Helper()

	switch {
	case opts.Nomos.MultiRepo:
		if opts.MultiRepoIncompatible {
			t.Skip("Test incompatible with MultiRepo mode")
		}

		switch *e2e.SkipMode {
		case e2e.RunDefault:
			if opts.SkipMultiRepo {
				t.Skip("Test skipped for MultiRepo mode")
			}
		case e2e.RunAll:
		case e2e.RunSkipped:
			if !opts.SkipMultiRepo {
				t.Skip("Test skipped for MultiRepo mode")
			}
		default:
			t.Fatalf("Invalid flag value %s for skipMode", *e2e.SkipMode)
		}
	case opts.SkipMonoRepo:
		t.Skip("Test skipped for MonoRepo mode")
	case len(opts.NamespaceRepos) > 0:
		// We're in MonoRepo mode and we aren't skipping this test, but there are
		// Namespace repos specified.
		t.Fatal("Namespace Repos specified, but running in MonoRepo mode. " +
			"Did you forget ntopts.SkipMonoRepo?")
	}

	if !*e2e.E2E {
		// This safeguard prevents test authors from accidentally forgetting to
		// check for the e2e test flag, so `go test ./...` functions as expected
		// without triggering e2e tests.
		t.Fatal("Attempting to createKindCluster cluster for non-e2e test. To fix, copy TestMain() from another e2e test.")
	}

	scheme := newScheme(t)
	c := connect(t, opts.RESTConfig, scheme)
	ctx := context.Background()

	nt := &NT{
		Context:                 ctx,
		T:                       t,
		ClusterName:             opts.Name,
		TmpDir:                  opts.TmpDir,
		Config:                  opts.RESTConfig,
		Client:                  c,
		kubeconfigPath:          filepath.Join(opts.TmpDir, ntopts.Kubeconfig),
		MultiRepo:               opts.Nomos.MultiRepo,
		FilesystemPollingPeriod: 50 * time.Millisecond,
		NonRootRepos:            make(map[string]*Repository),
		NamespaceRepos:          make(map[string]string),
		scheme:                  scheme,
	}

	if *e2e.ImagePrefix == e2e.DefaultImagePrefix {
		// We're using an ephemeral Kind cluster, so connect to the local Docker
		// repository. No need to clean before/after as these tests only exist for
		// a single test.
		connectToLocalRegistry(nt)
		docker.CheckImages(nt.T)
	} else {
		// We aren't using an ephemeral Kind cluster, so make sure the cluster is
		// clean before and after running the test.
		Clean(nt)
		t.Cleanup(func() {
			// Clean the cluster now that the test is over.
			Clean(nt)
		})
	}

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
	nt.gitPrivateKeyPath = generateSSHKeys(nt)

	t.Cleanup(func() {
		if t.Failed() {
			nt.testLogs(false)
			nt.testLogs(true)
			nt.testPods()
		}
	})
	waitForGit := installGitServer(nt)
	waitForConfigSync := installConfigSync(nt, opts.Nomos)

	err = waitForGit()
	if err != nil {
		t.Fatalf("waiting for git-server Deployment to become available: %v", err)
	}

	// allRepos specifies the slice all repos for portforwarding.
	allRepos := []string{rootRepo}
	for repo := range opts.MultiRepo.NamespaceRepos {
		allRepos = append(allRepos, repo)
	}

	// The git-server reports itself to be ready, so we don't have to wait on
	// anything.
	nt.gitRepoPort = portForwardGitServer(nt, allRepos...)
	nt.Root = NewRepository(nt, rootRepo, nt.TmpDir, nt.gitRepoPort, opts.SourceFormat)

	for nsr := range opts.MultiRepo.NamespaceRepos {
		nt.NonRootRepos[nsr] = NewRepository(nt, nsr, nt.TmpDir, nt.gitRepoPort, filesystem.SourceFormatUnstructured)
	}

	// First wait for CRDs to be established.
	if nt.MultiRepo {
		err = waitForCRDs(nt, multiRepoCRDs)
	} else {
		err = waitForCRDs(nt, monoRepoCRDs)
	}
	if err != nil {
		t.Fatalf("waiting for ConfigSync CRDs to become established: %v", err)
	}

	// ConfigSync custom types weren't available when the cluster was initially
	// created. Create a new Client, since it'll automatically be configured to
	// understand the Repo and RootSync types as ConfigSync is now installed.
	nt.RenewClient()

	err = waitForConfigSync(nt)
	if err != nil {
		t.Fatalf("waiting for ConfigSync Deployments to become available: %v", err)
	}

	switch opts.Control {
	case ntopts.DelegatedControl:
		setupDelegatedControl(nt, opts)
	case ntopts.CentralControl:
		setupCentralizedControl(nt, opts)
	default:
		// Most tests don't care about centralized/delegated control, but can
		// specify the behavior if that distinction is under test.
		setupCentralizedControl(nt, opts)
	}

	nt.WaitForRepoSyncs()
	return nt
}

// TestDir creates a unique temporary directory for the E2E test.
//
// Returned directory is absolute and OS-specific.
func TestDir(t *testing.T) string {
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
