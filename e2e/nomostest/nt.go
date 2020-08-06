package nomostest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

	// NonRootRepos is the Namespace repositories the cluster is syncing to.
	// Only used in multi-repo tests.
	NonRootRepos map[string]*Repository

	// gitPrivateKeyPath is the path to the private key used for communicating with the Git server.
	gitPrivateKeyPath string

	// gitRepoPort is the local port that forwards to the git repo deployment.
	gitRepoPort int

	// kubeconfigPath is the path to the kubeconfig file for the kind cluster
	kubeconfigPath string
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
	if obj.GetResourceVersion() != "" {
		// If obj is already populated, this can cause the final obj to be a
		// composite of multiple states of the object on the cluster.
		//
		// If this is due to a retry loop, remember to create a new instance to
		// populate for each loop.
		return errors.Errorf("called .Get on already-populated object")
	}
	return nt.Client.Get(nt.Context, client.ObjectKey{Name: name, Namespace: namespace}, obj)
}

// List is identical to List defined for client.Client, but without requiring Context.
func (nt *NT) List(obj runtime.Object, opts ...client.ListOption) error {
	return nt.Client.List(nt.Context, obj, opts...)
}

// Create is identical to Create defined for client.Client, but without requiring Context.
func (nt *NT) Create(obj core.Object, opts ...client.CreateOption) error {
	nt.T.Logf("creating %s", fmtObj(obj.GetName(), obj.GetNamespace(), obj))
	return nt.Client.Create(nt.Context, obj, opts...)
}

// Update is identical to Update defined for client.Client, but without requiring Context.
func (nt *NT) Update(obj core.Object, opts ...client.UpdateOption) error {
	nt.T.Logf("updating %s", fmtObj(obj.GetName(), obj.GetNamespace(), obj))
	return nt.Client.Update(nt.Context, obj, opts...)
}

// Delete is identical to Delete defined for client.Client, but without requiring Context.
func (nt *NT) Delete(obj core.Object, opts ...client.DeleteOption) error {
	nt.T.Logf("deleting %s", fmtObj(obj.GetName(), obj.GetNamespace(), obj))
	return nt.Client.Delete(nt.Context, obj, opts...)
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
		return errors.Errorf("%T %s/%s found", o, namespace, name)
	}
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// WaitForRepoSync is a convenience method that waits for the Repo's
// status.sync.latestToken is equal to the repository's sha1.
func (nt *NT) WaitForRepoSync() {
	nt.T.Helper()

	nt.WaitForSync(func() core.Object { return &v1.Repo{} }, "repo", RepoHasStatusSyncLatestToken)
}

// WaitForSync waits for the passed object to
// syncedTo is a function that accepts the sha1 hash of the repository and
// returns a Predicate that validates the passed object/name.
//
// Use WaitForRepoSync() if you're just waiting for ACM to report the latest
// commit has successfully synced.
//
// o returns a new object of the type to check is synced. It can't just be a
// struct pointer as calling .Get on the same struct pointer multiple times
// has undefined behavior.
func (nt *NT) WaitForSync(o func() core.Object, name string, syncedTo ...func(sha1 string) Predicate) {
	nt.T.Helper()

	sha1 := nt.Root.Hash()
	isSynced := make([]Predicate, len(syncedTo))
	for i, s := range syncedTo {
		isSynced[i] = s(sha1)
	}

	// Wait for the repository to report it is synced.
	took, err := Retry(120*time.Second, func() error {
		obj := o()
		return nt.Validate(name, "", obj, isSynced...)
	})
	if err != nil {
		nt.T.Logf("failed after %v to wait for sync", took)
		nt.T.Fatal(err)
	}
	nt.T.Logf("took %v to wait for sync", took)

	if _, isRepo := o().(*v1.Repo); isRepo {
		// Automatically renew the Client. We don't have tests that depend on behavior
		// when the test's client is out of date, and if ConfigSync reports that
		// everything has synced properly then (usually) new types should be available.
		nt.RenewClient()
	}
}

// RenewClient gets a new Client for talking to the cluster.
//
// Required whenever we expect the set of available types on the cluster
// to change. Called automatically at the end of WaitForSync.
//
// The only reason to call this manually from within a test is if we expect a
// controller to create a CRD dynamically, or if the test requires applying a
// CRD directly to the API Server.
func (nt *NT) RenewClient() {
	nt.T.Helper()

	nt.Client = connect(nt.T, nt.Config)
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
func (nt *NT) ApplyGatekeeperTestData(file string) {
	absPath := filepath.Join(baseDir, "e2e", "testdata", "gatekeeper", file)

	// We have to set validate=false because the default Gatekeeper YAMLs can't be
	// applied without it, and we aren't going to define our own compliant version.
	nt.Kubectl("apply", "-f", absPath, "--validate=false")
}
