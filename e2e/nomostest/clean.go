package nomostest

import (
	"fmt"
	"strings"
	"testing"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestLabel is the label added to all test objects, ensuring we can clean up
// non-ephemeral clusters when tests are complete.
const TestLabel = "nomos-test"

// TestLabelValue is the value assigned to the above label.
const TestLabelValue = "enabled"

// AddTestLabel is automatically added to objects created or declared with the
// NT methods, or declared with Repository.Add.
//
// This isn't perfect - objects added via other means (such as kubectl) will
// bypass this.
var AddTestLabel = core.Label(TestLabel, TestLabelValue)

// Clean removes all objects of types registered in the scheme, with the above
// caveats. It should be run before and after a test is run against any
// non-ephemeral cluster.
//
// It is unnecessary to run this on Kind clusters that exist only for the
// duration of a single test.
func Clean(nt *NT) {
	nt.T.Helper()

	// errDeleting ensures we delete everything possible to delete before failing.
	errDeleting := false
	for gvk := range nt.scheme.AllKnownTypes() {
		if !isListable(gvk.Kind) {
			continue
		}

		list, err := listType(nt, gvk)
		if err != nil {
			nt.T.Error(err)
			errDeleting = true
		}

		if len(list) > 0 && list[0].GetNamespace() != "" {
			// Ignore namespaced types.
			// It is much faster to delete Namespaces and have k8s automatically
			// delete namespaced resources inside.
			//
			// There isn't a quick way to delete many cluster-scoped resources, so
			// don't create tests that create thousands of cluster-scoped resources.
			continue
		}
		for _, u := range list {
			err := deleteObject(nt, &u)
			if err != nil {
				nt.T.Error(err)
				errDeleting = true
			}
		}
	}
	if errDeleting {
		nt.T.Fatal("error cleaning cluster")
	}

	// Now that we've told Kubernetes to delete everything, wait for it to be
	// deleted. We don't do this in the loop above for two reasons:
	//
	// 1) Waiting for every object individually to be deleted would take a long
	//    time.
	// 2) Some objects won't be deleted unless their dependencies are deleted
	//    first, and we can get stuck in a situation where finalizers haven't
	//    been removed.
	for gvk := range nt.scheme.AllKnownTypes() {
		if !isListable(gvk.Kind) {
			continue
		}

		list, err := listType(nt, gvk)
		if err != nil {
			nt.T.Error(err)
			errDeleting = true
		}

		if len(list) > 0 && list[0].GetNamespace() != "" {
			// Ignore namespaced types.
			// We're already blocking on waiting for the Namespaces to be deleted, so
			// waiting on a Namespaced type would do nothing.
			continue
		}

		for _, u := range list {
			WaitToTerminate(nt, u.GroupVersionKind(), u.GetName(), u.GetNamespace())
		}
	}
	if errDeleting {
		nt.T.Fatal("error waiting for test objects to be deleted")
	}
}

func listType(nt *NT, gvk schema.GroupVersionKind) ([]unstructured.Unstructured, error) {
	list := &unstructured.UnstructuredList{}
	list.GetObjectKind().SetGroupVersionKind(gvk)
	var opts []client.ListOption
	if gvk.Kind != "SyncList" {
		// For Syncs we can't rely on the TestLabel as this is generated by the
		// controller. We want all Syncs to be deleted.
		opts = append(opts, &client.MatchingLabels{TestLabel: TestLabelValue})
	}
	err := nt.Client.List(nt.Context, list, opts...)
	if err != nil && !meta.IsNoMatchError(err) && !apierrors.IsNotFound(err) {
		// Ignore cases where the type doesn't exist on the cluster. Obviously
		// there isn't anything to clean in that case.
		return nil, errors.Wrapf(err, "unable to clean objects of type %v from cluster", gvk)
	}

	return list.Items, nil
}

func deleteObject(nt *NT, u *unstructured.Unstructured) error {
	finalizers := u.GetFinalizers()
	if len(finalizers) == 1 && finalizers[0] == v1.SyncFinalizer {
		// Special case logic for the SyncFinalizer, as objects may get stuck
		// with it. We don't really care to wait for/rely on the controller to do
		// its cleanup as it may have exited in an error state.
		u.SetFinalizers([]string{})
		err := nt.Client.Update(nt.Context, u)
		switch {
		case apierrors.IsNotFound(err) || meta.IsNoMatchError(err):
			// The object isn't found, or the type no longer exists (in which case the
			// object definitely doesn't exist and we're done.
			return nil
		case err != nil:
			return errors.Wrapf(err, "unable to remove syncer finalizer from object %v: %s/%s",
				u.GroupVersionKind(), u.GetNamespace(), u.GetName())
		}
	}

	err := nt.Client.Delete(nt.Context, u)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "unable to clean test object from cluster %v: %s/%s",
			u.GroupVersionKind(), u.GetNamespace(), u.GetName())
	}

	return nil
}

func isListable(kind string) bool {
	// Only try to list types that have *List types associated with them, as they
	// are guaranteed to be listable.
	//
	// StatusList types are vestigial, have odd semantics, and are deprecated in 1.19.
	// Also we don't care about them for tests.
	return strings.HasSuffix(kind, "List") && !strings.HasSuffix(kind, "StatusList")
}

// FailIfUnknown fails the test if the passed type is not declared in the passed
// scheme.
func FailIfUnknown(t *testing.T, scheme *runtime.Scheme, o runtime.Object) {
	t.Helper()

	gvks, _, _ := scheme.ObjectKinds(o)
	if len(gvks) == 0 {
		t.Fatalf("unknown type %T %v. Add it to nomostest.newScheme().", o, o.GetObjectKind().GroupVersionKind())
	}
}

// WaitToTerminate waits for the passed object to be deleted.
// Immediately fails the test if the object is not deleted within the timeout.
func WaitToTerminate(nt *NT, gvk schema.GroupVersionKind, name, namespace string, opts ...WaitOption) {
	nt.T.Helper()

	Wait(nt.T, fmt.Sprintf("wait for %q %v to terminate", name, gvk), func() error {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(gvk)
		return nt.ValidateNotFound(name, namespace, u)
	}, opts...)
}
