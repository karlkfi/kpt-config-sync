package nomostest

import (
	"fmt"
	"strings"

	"github.com/google/nomos/e2e/nomostest/testing"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/webhook/configuration"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/common"
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

// FailOnError indicates whether the clean task should fail the test. If it is false, it only logs the failure without failing the test.
// The test should fail if the cleanup task fails before running a test.
// We tolerate the after-test cleanup failure as the before-test cleanup will guarantee the cluster is clean.
type FailOnError bool

// Clean removes all objects of types registered in the scheme, with the above
// caveats. It should be run before and after a test is run against any
// non-ephemeral cluster.
//
// It is unnecessary to run this on Kind clusters that exist only for the
// duration of a single test.
func Clean(nt *NT, failOnError FailOnError) {
	nt.T.Helper()

	// The admission-webhook prevents deleting test resources. Hence we delete it before cleaning other resources.
	removeAdmissionWebhook(nt, failOnError)

	// Reset any modified system namespaces.
	resetSystemNamespaces(nt, failOnError)

	// errDeleting ensures we delete everything possible to delete before failing.
	errDeleting := false
	for gvk := range nt.scheme.AllKnownTypes() {
		if !isListable(gvk.Kind) {
			continue
		}

		list, err := listType(nt, gvk)
		if err != nil {
			nt.T.Log(err)
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
				nt.T.Log(err)
				errDeleting = true
			}
		}
	}
	if errDeleting && bool(failOnError) {
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
			nt.T.Log(err)
			errDeleting = true
		}

		if len(list) > 0 && list[0].GetNamespace() != "" {
			// Ignore namespaced types.
			// We're already blocking on waiting for the Namespaces to be deleted, so
			// waiting on a Namespaced type would do nothing.
			continue
		}

		for _, u := range list {
			if failOnError {
				WaitToTerminate(nt, u.GroupVersionKind(), u.GetName(), u.GetNamespace())
			} else {
				WaitToTerminate(nt, u.GroupVersionKind(), u.GetName(), u.GetNamespace(), WaitNoFail())
			}
		}
	}
	if errDeleting && bool(failOnError) {
		nt.T.Fatal("error waiting for test objects to be deleted")
	}

	deleteImplicitNamespaces(nt, failOnError)
}

func isConfigSyncAnnotation(annotation string) bool {
	return annotation == common.LifecycleDeleteAnnotation ||
		strings.Contains(annotation, v1.ConfigManagementPrefix) ||
		strings.Contains(annotation, constants.ConfigSyncPrefix) ||
		annotation == applier.OwningInventoryKey ||
		annotation == hnc.AnnotationKeyV1A2
}

func isTestAnnotation(annotation string) bool {
	return strings.HasPrefix(annotation, "test-")
}

func isConfigSyncLabel(key, value string) bool {
	return (key == v1.ManagedByKey && value == v1.ManagedByValue) || key == configuration.DeclaredVersionLabel || strings.Contains(key, hnc.DepthSuffix)
}

func isTestLabel(key string) bool {
	return key == TestLabel || strings.HasPrefix(key, "test-")
}

// deleteConfigSyncAndTestAnnotationsAndLabels removes config sync and test annotations and labels from the namespace and update it.
func deleteConfigSyncAndTestAnnotationsAndLabels(nt *NT, ns *corev1.Namespace) error {
	var annotations = make(map[string]string)
	var labels = make(map[string]string)
	for k, v := range ns.Annotations {
		if !isConfigSyncAnnotation(k) && !isTestAnnotation(k) {
			annotations[k] = v
		}
	}
	for k, v := range ns.Labels {
		if !isConfigSyncLabel(k, v) && !isTestLabel(k) {
			labels[k] = v
		}
	}

	ns.Annotations = annotations
	ns.Labels = labels

	return nt.Client.Update(nt.Context, ns)
}

func namespaceHasNoConfigSyncAnnotationsAndLabels() Predicate {
	return func(o client.Object) error {
		ns, ok := o.(*corev1.Namespace)
		if !ok {
			return WrongTypeErr(ns, &corev1.Namespace{})
		}
		for k := range ns.Annotations {
			if isConfigSyncAnnotation(k) {
				return fmt.Errorf("found config sync annotation %s in namespace %s", k, ns.Name)
			}
		}
		for k, v := range ns.Labels {
			if isConfigSyncLabel(k, v) {
				return fmt.Errorf("found config sync label `%s: %s` in namespace %s", k, v, ns.Name)
			}
		}
		return nil
	}
}

// resetSystemNamespaces removes the config sync annotations and labels from the system namespaces.
func resetSystemNamespaces(nt *NT, failOnError FailOnError) {
	errDeleting := false
	nsList := &corev1.NamespaceList{}
	if err := nt.Client.List(nt.Context, nsList); err != nil {
		nt.T.Log(err)
		errDeleting = true
	}
	for _, ns := range nsList.Items {
		ns.SetGroupVersionKind(kinds.Namespace())
		if differ.IsManageableSystemNamespace(&ns) {
			if err := deleteConfigSyncAndTestAnnotationsAndLabels(nt, &ns); err != nil {
				nt.T.Log(err)
				errDeleting = true
			}
			if err := nt.Validate(ns.Name, "", &corev1.Namespace{}, namespaceHasNoConfigSyncAnnotationsAndLabels()); err != nil {
				if failOnError {
					nt.T.Fatal(err)
				} else {
					nt.T.Log(err)
				}
			}
		}
	}
	if errDeleting && bool(failOnError) {
		nt.T.Fatal("error resetting the system namespaces")
	}
}

// deleteImplicitNamespaces deletes the namespaces with the PreventDeletion Annotation.
func deleteImplicitNamespaces(nt *NT, failOnError FailOnError) {
	errDeleting := false
	nsList := &corev1.NamespaceList{}
	if err := nt.Client.List(nt.Context, nsList); err != nil {
		nt.T.Log(err)
		errDeleting = true
	}
	for _, ns := range nsList.Items {
		if annotation, ok := ns.Annotations[common.LifecycleDeleteAnnotation]; ok && annotation == common.PreventDeletion {
			if err := nt.Client.Delete(nt.Context, &ns); err != nil {
				nt.T.Log(err)
				errDeleting = true
			}
			if failOnError {
				WaitToTerminate(nt, kinds.Namespace(), ns.Name, "")
			} else {
				WaitToTerminate(nt, kinds.Namespace(), ns.Name, "", WaitNoFail())
			}
		}
	}
	if errDeleting && bool(failOnError) {
		nt.T.Fatal("error deleting implicit namespaces")
	}
}

// removeAdmissionWebhook deletes the `admission-webhook.configsync.gke.io` ValidatingWebhookConfiguration.
func removeAdmissionWebhook(nt *NT, failOnError FailOnError) {
	_, err := nt.Kubectl("delete", "validatingwebhookconfigurations", configuration.Name, "--ignore-not-found")
	if err != nil {
		if failOnError {
			nt.T.Fatal("error deleting admission-webhook")
		} else {
			nt.T.Log("error deleting admission-webhook")
		}
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
func FailIfUnknown(t testing.NTB, scheme *runtime.Scheme, o client.Object) {
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
