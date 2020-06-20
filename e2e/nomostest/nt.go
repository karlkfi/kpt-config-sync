package nomostest

import (
	"context"
	"fmt"
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

	// Name is the unique name of the test run.
	Name string

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

	// Repository is the repository the cluster is syncing to.
	// TODO(
	Repository *Repository

	// gitPrivateKeyPath is the path to the private key used for communicating with the Git server.
	gitPrivateKeyPath string

	// gitRepoPort is the local port that forwards to the git repo.
	gitRepoPort int

	// kubeconfigPath is the path to the kubeconfig file for the kind cluster
	kubeconfigPath string
}

// GitPrivateKeyPath returns the path to the git private key.
func (nt *NT) GitPrivateKeyPath() string {
	return nt.gitPrivateKeyPath
}

// GitRepoPort returns the path to the git private key.
func (nt *NT) GitRepoPort() int {
	return nt.gitRepoPort
}

// KubeconfigPath returns the path to the kubeconifg file.
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
	nt.T.Logf("getting %s", fmtObj(name, namespace, obj))
	return nt.Client.Get(nt.Context, client.ObjectKey{Name: name, Namespace: namespace}, obj)
}

// List is identical to List defined for client.Client, but without requiring Context.
func (nt *NT) List(obj runtime.Object, opts ...client.ListOption) error {
	nt.T.Logf("listing %T", obj)
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

// WaitForSync waits for the Repo object to report that the sync token on the
// Repo object matches the version in the Git repository.
func (nt *NT) WaitForSync() {
	nt.T.Helper()

	sha1 := nt.Repository.Hash()

	// Wait for the repository to report it is synced.
	err := Retry(60*time.Second, func() error {
		repo := &v1.Repo{}
		return nt.Validate("repo", "", repo, func(o core.Object) error {
			// Check the Sync.LatestToken as:
			// 1) Source.LatestToken is the most-recently-cloned hash of the git repository.
			//      It just means we've seen the update to the repository, but haven't
			//      updated the state of any objects on the cluster.
			// 2) Import.LatestToken is updated once we've successfully written the
			//      declared objects to ClusterConfigs/NamespaceConfigs, but haven't
			//      necessarily applied them to the cluster successfully.
			// 3) Sync.LatestToken is updated once we've updated the state of all
			//      objects on the cluster to match their declared states, so this is
			//      the one we want.
			if token := repo.Status.Sync.LatestToken; token != sha1 {
				return fmt.Errorf("status.sync.latestToken %q does not match git revision %q",
					token, sha1)
			}
			return nil
		})
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Automatically renew the Client. We don't have tests that depend on behavior
	// when the test's client is out of date, and if ConfigSync reports that
	// everything has synced properly then (usually) new types should be available.
	nt.Client = connect(nt.T, nt.Config)
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
