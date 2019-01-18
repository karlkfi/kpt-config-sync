package test

import (
	"reflect"
	"testing"

	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// StartTestManager begins a new test manager, and returns a function
// to gracefully shutdown.
func StartTestManager(t *testing.T, cfg *rest.Config) (manager.Manager, func()) {
	t.Helper()
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		t.Fatalf("error creating manager: %v", err)
	}
	stop := make(chan struct{})
	go func() {
		if err := mgr.Start(stop); err != nil {
			glog.Warningf("unable to start manager: %v", err)
		}
	}()
	return mgr, func() {
		close(stop)
	}
}

// SyncCache forces the test manager to update its cache and blocks until it has completed.
func SyncCache(t *testing.T, mgr manager.Manager) {
	t.Helper()
	cache := mgr.GetCache()
	ch := make(chan struct{})
	defer close(ch)
	if !cache.WaitForCacheSync(ch) {
		t.Fatalf("could not sync cache")
	}
}

// RunReconcilerAssertResults asserts the expected state of the reconciler run.
func RunReconcilerAssertResults(t *testing.T, reconciler reconcile.Reconciler, objectMeta v1.ObjectMeta,
	expectedResult reconcile.Result, expectedError error) {
	t.Helper()
	name := types.NamespacedName{Namespace: objectMeta.Namespace, Name: objectMeta.Name}
	reconcileRequest := reconcile.Request{NamespacedName: name}
	result, err := reconciler.Reconcile(reconcileRequest)
	if expectedError == nil {
		if err != nil {
			t.Fatalf("reconcile returned unexpected error: %v", err)
		}
	} else {
		if err == nil || err.Error() != expectedError.Error() {
			t.Fatalf("error mismatch: got '%v', want '%v'", err, expectedError)
		}
	}
	if !reflect.DeepEqual(expectedResult, result) {
		t.Fatalf("reconcile result mismatch: got '%v', want '%v'", result, expectedResult)
	}
}
