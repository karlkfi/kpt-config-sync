package nomostest

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

	// Client is the underlying client used to talk to the Kubernetes cluster.
	//
	// Most tests shouldn't need to talk directly to this, unless simulating
	// direct interactions with the API Server.
	Client client.Client
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
