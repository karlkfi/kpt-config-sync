package nomostest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NT represents the test environment for a single Nomos end-to-end test case.
type NT struct {
	Context context.Context

	// T is the test environment for the test.
	// Used to exit tests early when setup fails, and for logging.
	T *testing.T

	// ClusterName is the unique name of the test run.
	ClusterName string

	// TmpDir is the temporary directory the test will write to.
	// By default, automatically deleted when the test finishes.
	TmpDir string

	// Config specifies how to create a new connection to the cluster.
	Config *rest.Config

	// Client is the underlying client used to talk to the Kubernetes cluster.
	//
	// Most tests shouldn't need to talk directly to this, unless simulating
	// direct interactions with the API Server.
	Client client.Client

	// Root is the root repository the cluster is syncing to.
	Root *Repository

	// MultiRepo indicates that the test case is for multi-repo Config Sync.
	MultiRepo bool

	// NonRootRepos is the Namespace repositories the cluster is syncing to.
	// Only used in multi-repo tests.
	NonRootRepos map[string]*Repository

	// NamespaceRepos is a map from Namespace names to the name of the Repository
	// containing configs for that Namespace.
	NamespaceRepos map[string]string

	// FilesystemPollingPeriod is the time between checking the filessystem for udpates
	// to the local Git repository.
	FilesystemPollingPeriod time.Duration

	// gitPrivateKeyPath is the path to the private key used for communicating with the Git server.
	gitPrivateKeyPath string

	// gitRepoPort is the local port that forwards to the git repo deployment.
	gitRepoPort int

	// kubeconfigPath is the path to the kubeconfig file for the kind cluster
	kubeconfigPath string

	// scheme is the Scheme for the test suite that maps from structs to GVKs.
	scheme *runtime.Scheme
}

// GitPrivateKeyPath returns the path to the git private key.
//
// Deprecated: only the legacy bats tests should make use of this function.
func (nt *NT) GitPrivateKeyPath() string {
	return nt.gitPrivateKeyPath
}

// GitRepoPort returns the path to the git private key.
//
// Deprecated: only the legacy bats tests should make use of this function.
func (nt *NT) GitRepoPort() int {
	return nt.gitRepoPort
}

// KubeconfigPath returns the path to the kubeconifg file.
//
// Deprecated: only the legacy bats tests should make use of this function.
func (nt *NT) KubeconfigPath() string {
	return nt.kubeconfigPath
}

func fmtObj(name, namespace string, obj runtime.Object) string {
	return fmt.Sprintf("%s/%s %T", namespace, name, obj)
}

// Get is identical to Get defined for client.Client, except:
//
// 1) Context implicitly uses the one created for the test case.
// 2) name and namespace are strings instead of requiring client.ObjectKey.
//
// Leave namespace as empty string for cluster-scoped resources.
func (nt *NT) Get(name, namespace string, obj core.Object) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	if obj.GetResourceVersion() != "" {
		// If obj is already populated, this can cause the final obj to be a
		// composite of multiple states of the object on the cluster.
		//
		// If this is due to a retry loop, remember to create a new instance to
		// populate for each loop.
		return errors.Errorf("called .Get on already-populated object %v: %v", obj.GroupVersionKind(), obj)
	}
	return nt.Client.Get(nt.Context, client.ObjectKey{Name: name, Namespace: namespace}, obj)
}

// List is identical to List defined for client.Client, but without requiring Context.
func (nt *NT) List(obj runtime.Object, opts ...client.ListOption) error {
	return nt.Client.List(nt.Context, obj, opts...)
}

// Create is identical to Create defined for client.Client, but without requiring Context.
func (nt *NT) Create(obj core.Object, opts ...client.CreateOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.T.Logf("creating %s", fmtObj(obj.GetName(), obj.GetNamespace(), obj))
	AddTestLabel(obj)
	return nt.Client.Create(nt.Context, obj, opts...)
}

// Update is identical to Update defined for client.Client, but without requiring Context.
func (nt *NT) Update(obj core.Object, opts ...client.UpdateOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.T.Logf("updating %s", fmtObj(obj.GetName(), obj.GetNamespace(), obj))
	return nt.Client.Update(nt.Context, obj, opts...)
}

// Delete is identical to Delete defined for client.Client, but without requiring Context.
func (nt *NT) Delete(obj core.Object, opts ...client.DeleteOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.T.Logf("deleting %s", fmtObj(obj.GetName(), obj.GetNamespace(), obj))
	return nt.Client.Delete(nt.Context, obj, opts...)
}

// MergePatch uses the object to construct a merge patch for the fields provided.
func (nt *NT) MergePatch(obj core.Object, patch string, opts ...client.PatchOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.T.Logf("Applying patch %s", patch)
	AddTestLabel(obj)
	return nt.Client.Patch(nt.Context, obj, client.ConstantPatch(types.MergePatchType, []byte(patch)), opts...)
}

// MustMergePatch is like MergePatch but will call t.Fatal if the patch fails.
func (nt *NT) MustMergePatch(obj core.Object, patch string, opts ...client.PatchOption) {
	nt.T.Helper()
	if err := nt.MergePatch(obj, patch, opts...); err != nil {
		nt.T.Fatal(err)
	}
}

// Validate returns an error if the indicated object does not exist.
//
// Validates the object against each of the passed Predicates, returning error
// if any Predicate fails.
func (nt *NT) Validate(name, namespace string, o core.Object, predicates ...Predicate) error {
	err := nt.Get(name, namespace, o)
	if err != nil {
		return err
	}
	for _, p := range predicates {
		err = p(o)
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidateNotFound returns an error if the indicated object exists.
//
// `o` must either be:
// 1) a struct pointer to the type of the object to search for, or
// 2) an unstructured.Unstructured with the type information filled in.
func (nt *NT) ValidateNotFound(name, namespace string, o core.Object) error {
	err := nt.Get(name, namespace, o)
	if err == nil {
		return errors.Errorf("%T %v %s/%s found", o, o.GroupVersionKind(), namespace, name)
	}
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// WaitForRepoSyncs is a convenience method that waits for all repositories
// to sync.
//
// Unless you're testing pre-CSMR functionality related to in-cluster objects,
// you should be using this function to block on ConfigSync to sync everything.
//
// If you want to check the internals of specific objects (e.g. the error field
// of a RepoSync), use nt.Validate() - possibly in a Retry.
func (nt *NT) WaitForRepoSyncs() {
	nt.T.Helper()

	if nt.MultiRepo {
		nt.WaitForRootSync(func() core.Object { return &v1alpha1.RootSync{} },
			"root-sync", configmanagement.ControllerNamespace, RootSyncHasStatusSyncCommit)

		for ns, repo := range nt.NamespaceRepos {
			nt.WaitForRepoSync(repo, func() core.Object { return &v1alpha1.RepoSync{} },
				v1alpha1.RepoSyncName, ns, RepoSyncHasStatusSyncCommit)
		}
	} else {
		nt.WaitForRootSync(func() core.Object { return &v1.Repo{} },
			"repo", "", RepoHasStatusSyncLatestToken)
	}
}

// WaitForRepoSync waits for the specified RepoSync to be synced to HEAD
// of the specified repository.
func (nt *NT) WaitForRepoSync(repoName string, o func() core.Object,
	name, namespace string, syncedTo func(sha1 string) Predicate) {
	nt.T.Helper()

	// Get the repository this RepoSync is syncing to, and ensure it is synced
	// to HEAD.
	repo, exists := nt.NonRootRepos[repoName]
	if !exists {
		nt.T.Fatal("checked if nonexistent repo is synced")
	}
	sha1 := repo.Hash()
	nt.waitForSync(o, name, namespace, syncedTo(sha1))
}

// waitForSync waits for the specified object to be synced.
//
// o returns a new object of the type to check is synced. It can't just be a
// struct pointer as calling .Get on the same struct pointer multiple times
// has undefined behavior.
//
// name and namespace identify the specific object to check.
//
// predicates is a list of Predicates to use to tell whether the object is
// synced as desired.
func (nt *NT) waitForSync(o func() core.Object,
	name, namespace string, predicates ...Predicate) {
	nt.T.Helper()

	// Wait for the repository to report it is synced.
	took, err := Retry(120*time.Second, func() error {
		obj := o()
		return nt.Validate(name, namespace, obj, predicates...)
	})
	if err != nil {
		nt.T.Logf("failed after %v to wait for sync", took)
		nt.T.Fatal(err)
	}
	nt.T.Logf("took %v to wait for sync", took)

	// Automatically renew the Client. We don't have tests that depend on behavior
	// when the test's client is out of date, and if ConfigSync reports that
	// everything has synced properly then (usually) new types should be available.
	if _, isRepo := o().(*v1.Repo); isRepo {
		nt.RenewClient()
	}
	if _, isRepoSync := o().(*v1alpha1.RepoSync); isRepoSync {
		nt.RenewClient()
	}
}

// WaitForRootSync waits for the specified object to be synced to the root
// repository.
//
// o returns a new struct pointer to the desired object type each time it is
// called.
//
// name and namespace uniquely specify an object of the desired type.
//
// syncedTo specify how to check that the object is synced to HEAD. This function
// automatically checks for HEAD of the root repository.
func (nt *NT) WaitForRootSync(o func() core.Object, name, namespace string, syncedTo ...func(sha1 string) Predicate) {
	nt.T.Helper()

	sha1 := nt.Root.Hash()
	isSynced := make([]Predicate, len(syncedTo))
	for i, s := range syncedTo {
		isSynced[i] = s(sha1)
	}
	nt.waitForSync(o, name, namespace, isSynced...)
}

// RenewClient gets a new Client for talking to the cluster.
//
// Required whenever we expect the set of available types on the cluster
// to change. Called automatically at the end of WaitForRootSync.
//
// The only reason to call this manually from within a test is if we expect a
// controller to create a CRD dynamically, or if the test requires applying a
// CRD directly to the API Server.
func (nt *NT) RenewClient() {
	nt.T.Helper()

	nt.Client = connect(nt.T, nt.Config, nt.scheme)
}

// Kubectl is a convenience method for calling kubectl against the
// currently-connected cluster.
//
// Fails the test if the kubectl command fails - call kubectl directly if you
// want specialized behavior.
func (nt *NT) Kubectl(args ...string) {
	nt.T.Helper()

	prefix := []string{"--kubeconfig", nt.kubeconfigPath}
	args = append(prefix, args...)
	out, err := exec.Command("kubectl", args...).CombinedOutput()
	if err != nil {
		nt.T.Log(append([]string{"kubectl"}, args...))
		nt.T.Log(string(out))
		nt.T.Fatal(err)
	}
}

// ApplyGatekeeperTestData is an exception to the "all test data is specified inline"
// rule. It isn't informative to literally have the CRD specifications in the
// test code, and we have strict type requirements on how the CRD is laid out.
func (nt *NT) ApplyGatekeeperTestData(file, crd string) error {
	absPath := filepath.Join(baseDir, "e2e", "testdata", "gatekeeper", file)

	// We have to set validate=false because the default Gatekeeper YAMLs can't be
	// applied without it, and we aren't going to define our own compliant version.
	nt.Kubectl("apply", "-f", absPath, "--validate=false")
	err := waitForCRDs(nt, []string{crd})
	if err != nil {
		nt.RenewClient()
	}
	return err
}

// WaitForRootSyncSourceErrorCode waits until the given error code is present on the RootSync resource
func (nt *NT) WaitForRootSyncSourceErrorCode(code string, opts ...WaitOption) {
	Wait(nt.T, fmt.Sprintf("RootSync error code %s", code),
		func() error {
			rs := fake.RootSyncObject()
			err := nt.Get(rs.GetName(), rs.GetNamespace(), rs)
			if err != nil {
				return err
			}
			errs := rs.Status.Source.Errors
			if len(errs) == 0 {
				return errors.Errorf("no errors present")
			}
			var codes []string
			for _, e := range errs {
				if e.Code == code {
					return nil
				}
				codes = append(codes, e.Code)
			}
			return errors.Errorf("error %s not present, got %s", code, strings.Join(codes, ", "))
		},
		opts...,
	)
}

// WaitForRootSyncSourceErrorClear waits until the given error code is present on the RootSync resource
func (nt *NT) WaitForRootSyncSourceErrorClear(opts ...WaitOption) {
	Wait(nt.T, "RootSync errors cleared",
		func() error {
			rs := fake.RootSyncObject()
			err := nt.Get(rs.GetName(), rs.GetNamespace(), rs)
			if err != nil {
				return err
			}
			errs := rs.Status.Source.Errors
			if len(errs) == 0 {
				return nil
			}
			var messages []string
			for _, e := range errs {
				messages = append(messages, e.ErrorMessage)
			}
			return errors.Errorf("got errors %s", strings.Join(messages, ", "))
		},
		opts...,
	)
}

// WaitOption is an optional parameter for Wait
type WaitOption func(wait *waitSpec)

type waitSpec struct {
	timeout time.Duration
}

// WaitTimeout provides the timeout option to Wait.
func WaitTimeout(timeout time.Duration) WaitOption {
	return func(wait *waitSpec) {
		wait.timeout = timeout
	}
}

// Wait provides a logged wait for condition to return nil with options for timeout.
func Wait(t *testing.T, opName string, condition func() error, opts ...WaitOption) {
	t.Helper()

	wait := waitSpec{
		timeout: time.Second * 120,
	}
	for _, opt := range opts {
		opt(&wait)
	}

	// Wait for the repository to report it is synced.
	took, err := Retry(wait.timeout, condition)
	if err != nil {
		t.Logf("failed after %v to wait for %s", took, opName)
		t.Fatal(err)
	}
	t.Logf("took %v to wait for %s", took, opName)
}
