package nomostest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/nomos/e2e"
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func fmtObj(obj core.Object) string {
	return fmt.Sprintf("%s/%s %T", obj.GetNamespace(), obj.GetName(), obj)
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
func (nt *NT) List(obj client.ObjectList, opts ...client.ListOption) error {
	return nt.Client.List(nt.Context, obj, opts...)
}

// Create is identical to Create defined for client.Client, but without requiring Context.
func (nt *NT) Create(obj core.Object, opts ...client.CreateOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.DebugLogf("creating %s", fmtObj(obj))
	AddTestLabel(obj)
	return nt.Client.Create(nt.Context, obj, opts...)
}

// Update is identical to Update defined for client.Client, but without requiring Context.
func (nt *NT) Update(obj core.Object, opts ...client.UpdateOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.DebugLogf("updating %s", fmtObj(obj))
	return nt.Client.Update(nt.Context, obj, opts...)
}

// Delete is identical to Delete defined for client.Client, but without requiring Context.
func (nt *NT) Delete(obj core.Object, opts ...client.DeleteOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.DebugLogf("deleting %s", fmtObj(obj))
	return nt.Client.Delete(nt.Context, obj, opts...)
}

// DeleteAllOf is identical to DeleteAllOf defined for client.Client, but without requiring Context.
func (nt *NT) DeleteAllOf(obj core.Object, opts ...client.DeleteAllOfOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.DebugLogf("deleting all of %T", obj)
	return nt.Client.DeleteAllOf(nt.Context, obj, opts...)
}

// MergePatch uses the object to construct a merge patch for the fields provided.
func (nt *NT) MergePatch(obj core.Object, patch string, opts ...client.PatchOption) error {
	FailIfUnknown(nt.T, nt.scheme, obj)
	nt.DebugLogf("Applying patch %s", patch)
	AddTestLabel(obj)
	return nt.Client.Patch(nt.Context, obj, client.RawPatch(types.MergePatchType, []byte(patch)), opts...)
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
		nt.WaitForRootSync(kinds.RootSync(),
			"root-sync", configmanagement.ControllerNamespace, RootSyncHasStatusSyncCommit)

		for ns, repo := range nt.NamespaceRepos {
			nt.WaitForRepoSync(repo, kinds.RepoSync(),
				v1alpha1.RepoSyncName, ns, RepoSyncHasStatusSyncCommit)
		}
	} else {
		nt.WaitForRootSync(kinds.Repo(),
			"repo", "", RepoHasStatusSyncLatestToken)
	}
}

// WaitForRepoSync waits for the specified RepoSync to be synced to HEAD
// of the specified repository.
func (nt *NT) WaitForRepoSync(repoName string, gvk schema.GroupVersionKind,
	name, namespace string, syncedTo func(sha1 string) Predicate) {
	nt.T.Helper()

	// Get the repository this RepoSync is syncing to, and ensure it is synced
	// to HEAD.
	repo, exists := nt.NonRootRepos[repoName]
	if !exists {
		nt.T.Fatal("checked if nonexistent repo is synced")
	}
	sha1 := repo.Hash()
	nt.waitForSync(gvk, name, namespace, syncedTo(sha1))
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
func (nt *NT) waitForSync(gvk schema.GroupVersionKind, name, namespace string, predicates ...Predicate) {
	nt.T.Helper()

	// Wait for the repository to report it is synced.
	took, err := Retry(120*time.Second, func() error {
		obj, err := nt.scheme.New(gvk)
		if err != nil {
			return fmt.Errorf("%w: got unrecognized GVK %v", ErrWrongType, gvk)
		}
		o, ok := obj.(core.Object)
		if !ok {
			// This means the GVK corresponded to a type registered in the Scheme
			// which is not a valid Kubernetes object. We expect the only way this
			// can happen is if gvk is for a List type, like NamespaceList.
			return errors.Wrapf(ErrWrongType, "trying to wait for List type to sync: %T", o)
		}

		return nt.Validate(name, namespace, o, predicates...)
	})
	if err != nil {
		nt.T.Logf("failed after %v to wait for %s/%s %v to be synced", took, namespace, name, gvk)

		nt.T.Fatal(err)
	}
	nt.T.Logf("took %v to wait for %s/%s %v to be synced", took, namespace, name, gvk)

	// Automatically renew the Client. We don't have tests that depend on behavior
	// when the test's client is out of date, and if ConfigSync reports that
	// everything has synced properly then (usually) new types should be available.
	if gvk == kinds.Repo() || gvk == kinds.RepoSync() {
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
func (nt *NT) WaitForRootSync(gvk schema.GroupVersionKind, name, namespace string, syncedTo ...func(sha1 string) Predicate) {
	nt.T.Helper()

	sha1 := nt.Root.Hash()
	isSynced := make([]Predicate, len(syncedTo))
	for i, s := range syncedTo {
		isSynced[i] = s(sha1)
	}
	nt.waitForSync(gvk, name, namespace, isSynced...)
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
// currently-connected cluster. Returns STDOUT, and an error if kubectl exited
// abnormally.
//
// If you want to fail the test immediately on failure, use MustKubectl.
func (nt *NT) Kubectl(args ...string) ([]byte, error) {
	nt.T.Helper()

	prefix := []string{"--kubeconfig", nt.kubeconfigPath}
	args = append(prefix, args...)
	nt.DebugLogf("kubectl %s", strings.Join(args, " "))
	return exec.Command("kubectl", args...).CombinedOutput()
}

// MustKubectl fails the test immediately if the kubectl command fails. On
// success, returns STDOUT.
func (nt *NT) MustKubectl(args ...string) []byte {
	out, err := nt.Kubectl(args...)
	if err != nil {
		nt.T.Log(append([]string{"kubectl"}, args...))
		nt.T.Log(string(out))
		nt.T.Fatal(err)
	}
	return out
}

// DebugLog is like nt.T.Log, but only prints the message if --debug is passed.
// Use for fine-grained information that is unlikely to cause failures in CI.
func (nt *NT) DebugLog(args ...interface{}) {
	if *e2e.Debug {
		nt.T.Log(args...)
	}
}

// DebugLogf is like nt.T.Logf, but only prints the message if --debug is passed.
// Use for fine-grained information that is unlikely to cause failures in CI.
func (nt *NT) DebugLogf(format string, args ...interface{}) {
	if *e2e.Debug {
		nt.T.Logf(format, args...)
	}
}

// PodLogs prints the logs from the specified deployment.
// If there is an error getting the logs for the specified deployment, prints
// the error.
func (nt *NT) PodLogs(namespace, deployment, container string, previousPodLog bool) {
	args := []string{"logs", fmt.Sprintf("deployment/%s", deployment), "-n", namespace}
	if previousPodLog {
		args = append(args, "-p")
	}
	if container != "" {
		args = append(args, container)
	}
	out, err := nt.Kubectl(args...)
	// Print a standardized header before each printed log to make ctrl+F-ing the
	// log you want easier.
	cmd := fmt.Sprintf("kubectl %s", strings.Join(args, " "))
	if err != nil {
		nt.T.Logf("failed to run %q: %v", cmd, err)
		return
	}
	nt.T.Logf("%s\n%s", cmd, string(out))
}

// testLogs print the logs for the current container instances when `previousPodLog` is false.
// testLogs print the logs for the previous container instances if they exist when `previousPodLog` is true.
func (nt *NT) testLogs(previousPodLog bool) {
	// These pods/containers are noisy and rarely contain useful information:
	// - git-sync
	// - fs-watcher
	// - monitor
	// Don't merge with any of these uncommented, but feel free to uncomment
	// temporarily to see how presubmit responds.
	if nt.MultiRepo {
		nt.PodLogs(configmanagement.ControllerNamespace, reconcilermanager.ManagerName, reconcilermanager.ManagerName, previousPodLog)
		nt.PodLogs(configmanagement.ControllerNamespace, reconciler.RootSyncName, reconcilermanager.Reconciler, previousPodLog)
		//nt.PodLogs(configmanagement.ControllerNamespace, reconcilermanager.RootSyncName, reconcilermanager.GitSync, previousPodLog)
		for ns := range nt.NamespaceRepos {
			nt.PodLogs(configmanagement.ControllerNamespace, reconciler.RepoSyncName(ns),
				reconcilermanager.Reconciler, previousPodLog)
			//nt.PodLogs(configmanagement.ControllerNamespace, reconcilermanager.RepoSyncName(ns), reconcilermanager.GitSync, previousPodLog)
		}
	} else {
		nt.PodLogs(configmanagement.ControllerNamespace, filesystem.GitImporterName, importer.Name, previousPodLog)
		//nt.PodLogs(configmanagement.ControllerNamespace, filesystem.GitImporterName, "fs-watcher", previousPodLog)
		//nt.PodLogs(configmanagement.ControllerNamespace, filesystem.GitImporterName, reconcilermanager.GitSync, previousPodLog)
		//nt.PodLogs(configmanagement.ControllerNamespace, state.MonitorName, "", previousPodLog)
	}
}

// testPods prints the output of `kubectl get pods`, which includes a 'RESTARTS' column
// indicating how many times each pod has restarted. If a pod has restarted, the following
// two commands can be used to get more information:
//   1) kubectl get pods -n config-management-system -o yaml
//   2) kubectl logs deployment/<deploy-name> <container-name> -n config-management-system -p
func (nt *NT) testPods() {
	out, err := nt.Kubectl("get", "pods", "-n", configmanagement.ControllerNamespace)
	// Print a standardized header before each printed log to make ctrl+F-ing the
	// log you want easier.
	nt.T.Logf("kubectl get pods -n %s: \n%s", configmanagement.ControllerNamespace, string(out))
	if err != nil {
		nt.T.Log("error running `kubectl get pods`:", err)
	}
}

// ApplyGatekeeperTestData is an exception to the "all test data is specified inline"
// rule. It isn't informative to literally have the CRD specifications in the
// test code, and we have strict type requirements on how the CRD is laid out.
func (nt *NT) ApplyGatekeeperTestData(file, crd string) error {
	absPath := filepath.Join(baseDir, "e2e", "testdata", "gatekeeper", file)

	// We have to set validate=false because the default Gatekeeper YAMLs can't be
	// applied without it, and we aren't going to define our own compliant version.
	nt.MustKubectl("apply", "-f", absPath, "--validate=false")
	err := waitForCRDs(nt, []string{crd})
	if err != nil {
		nt.RenewClient()
	}
	return err
}

func validateError(errs []v1alpha1.ConfigSyncError, code string) error {
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
}

func validateErrorClear(errs []v1alpha1.ConfigSyncError) error {
	if len(errs) == 0 {
		return nil
	}
	var messages []string
	for _, e := range errs {
		messages = append(messages, e.ErrorMessage)
	}
	return errors.Errorf("got errors %s", strings.Join(messages, ", "))
}

// WaitForRootSyncSourceError waits until the given error (code and message) is present on the RootSync resource
func (nt *NT) WaitForRootSyncSourceError(code string, opts ...WaitOption) {
	Wait(nt.T, fmt.Sprintf("RootSync source error code %s", code),
		func() error {
			rs := fake.RootSyncObject()
			err := nt.Get(rs.GetName(), rs.GetNamespace(), rs)
			if err != nil {
				return err
			}
			return validateError(rs.Status.Source.Errors, code)
		},
		opts...,
	)
}

// WaitForRootSyncSourceErrorClear waits until the given error code disappears from the RootSync resource
func (nt *NT) WaitForRootSyncSourceErrorClear(opts ...WaitOption) {
	Wait(nt.T, "RootSync source errors cleared",
		func() error {
			rs := fake.RootSyncObject()
			err := nt.Get(rs.GetName(), rs.GetNamespace(), rs)
			if err != nil {
				return err
			}
			return validateErrorClear(rs.Status.Source.Errors)
		},
		opts...,
	)
}

// WaitForRepoSyncSourceError waits until the given error (code and message) is present on the RepoSync resource
func (nt *NT) WaitForRepoSyncSourceError(namespace, code string, opts ...WaitOption) {
	Wait(nt.T, fmt.Sprintf("RepoSync source error code %s", code),
		func() error {
			rs := fake.RepoSyncObject(core.Namespace(namespace))
			err := nt.Get(rs.GetName(), rs.GetNamespace(), rs)
			if err != nil {
				return err
			}
			return validateError(rs.Status.Source.Errors, code)
		},
		opts...,
	)
}

// WaitForRepoSyncSourceErrorClear waits until the given error code disappears from the RepoSync resource
func (nt *NT) WaitForRepoSyncSourceErrorClear(namespace string, opts ...WaitOption) {
	Wait(nt.T, "RepoSync source errors cleared",
		func() error {
			rs := fake.RepoSyncObject(core.Namespace(namespace))
			err := nt.Get(rs.GetName(), rs.GetNamespace(), rs)
			if err != nil {
				return err
			}
			return validateErrorClear(rs.Status.Source.Errors)
		},
		opts...,
	)
}

// WaitForRepoSourceError waits until the given error (code and message) is present on the Repo resource
func (nt *NT) WaitForRepoSourceError(code string, opts ...WaitOption) {
	Wait(nt.T, fmt.Sprintf("Repo source error code %s", code),
		func() error {
			repo := &v1.Repo{}
			err := nt.Get("repo", "", repo)
			if err != nil {
				return err
			}
			errs := repo.Status.Source.Errors
			if len(errs) == 0 {
				return errors.Errorf("no errors present")
			}
			var codes []string
			for _, e := range errs {
				if e.Code == fmt.Sprintf("KNV%s", code) {
					return nil
				}
				codes = append(codes, e.Code)
			}
			return errors.Errorf("error %s not present, got %s", code, strings.Join(codes, ", "))
		},
		opts...,
	)
}

// WaitForRepoSourceErrorClear waits until the given error code disappears from the Repo resource
func (nt *NT) WaitForRepoSourceErrorClear(opts ...WaitOption) {
	Wait(nt.T, "Repo source errors cleared",
		func() error {
			repo := &v1.Repo{}
			err := nt.Get("repo", "", repo)
			if err != nil {
				return err
			}
			errs := repo.Status.Source.Errors
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

// WaitForRepoImportErrorCode waits until the given error code is present on the Repo resource.
func (nt *NT) WaitForRepoImportErrorCode(code string, opts ...WaitOption) {
	Wait(nt.T, fmt.Sprintf("Repo import error code %s", code),
		func() error {
			r := &v1.Repo{}
			err := nt.Get("repo", "", r)

			if err != nil {
				return err
			}
			errs := r.Status.Import.Errors
			if len(errs) == 0 {
				return errors.Errorf("no errors present")
			}
			var codes []string
			for _, e := range errs {
				if e.Code == fmt.Sprintf("KNV%s", code) {
					return nil
				}
				codes = append(codes, e.Code)
			}
			return errors.Errorf("error %s not present, got %s", code, strings.Join(codes, ", "))
		},
		opts...,
	)
}

// WaitOption is an optional parameter for Wait
type WaitOption func(wait *waitSpec)

type waitSpec struct {
	timeout time.Duration
	// failOnError is the flag to control whether to fail the test or not when errors occur.
	failOnError bool
}

// WaitTimeout provides the timeout option to Wait.
func WaitTimeout(timeout time.Duration) WaitOption {
	return func(wait *waitSpec) {
		wait.timeout = timeout
	}
}

// WaitNoFail sets failOnError to false so the Wait function only logs the error but not fails the test.
func WaitNoFail() WaitOption {
	return func(wait *waitSpec) {
		wait.failOnError = false
	}
}

// Wait provides a logged wait for condition to return nil with options for timeout.
// It fails the test on errors.
func Wait(t *testing.T, opName string, condition func() error, opts ...WaitOption) {
	t.Helper()

	wait := waitSpec{
		timeout:     time.Second * 120,
		failOnError: true,
	}
	for _, opt := range opts {
		opt(&wait)
	}

	// Wait for the repository to report it is synced.
	took, err := Retry(wait.timeout, condition)
	if err != nil {
		t.Logf("failed after %v to wait for %s", took, opName)
		if wait.failOnError {
			t.Fatal(err)
		}
	}
	t.Logf("took %v to wait for %s", took, opName)
}
