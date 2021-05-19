package e2e

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
)

// The reason we have both TestKubectlCreatesManagedNamespaceResourceMonoRepo and
// TestKubectlCreatesManagedNamespaceResourceMultiRepo is that the mono-repo mode and
// CSMR handles managed namespaces which are created by other parties differently:
//   * the mono-repo mode does not remove these namespaces;
//   * CSMR does remove these namespaces.

func TestKubectlCreatesManagedNamespaceResourceMonoRepo(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMultiRepo, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	ns := []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: test-ns
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _namespace_test-ns
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-ns.yaml"), ns, 0644); err != nil {
		t.Fatalf("failed to create a tmp file %v", err)
	}

	out, err := nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		t.Fatalf("got `kubectl apply -f test-ns.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the reconciler can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the namespace.
	err = nt.Validate("test-ns", "", &corev1.Namespace{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		t.Fatal(err)
	}

	ns = []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: test-ns
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _namespace_wrong-ns
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-ns.yaml"), ns, 0644); err != nil {
		t.Fatalf("failed to create a tmp file %v", err)
	}

	out, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		t.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the reconciler can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the namespace, since its `configsync.gke.io/resource-id`
	// annotation is incorrect.
	err = nt.Validate("test-ns", "", &corev1.Namespace{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestKubectlCreatesManagedNamespaceResourceMultiRepo(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	ns := []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: test-ns
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _namespace_test-ns
    configsync.gke.io/manager: :root
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-ns.yaml"), ns, 0644); err != nil {
		t.Fatalf("failed to create a tmp file %v", err)
	}

	out, err := nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		t.Fatalf("got `kubectl apply -f test-ns.yaml` error %v %s, want return nil", err, out)
	}

	// Config Sync should remove `test-ns`.
	nomostest.WaitToTerminate(nt, kinds.Namespace(), "test-ns", "")

	ns = []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: test-ns
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _namespace_wrong-ns
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-ns.yaml"), ns, 0644); err != nil {
		t.Fatalf("failed to create a tmp file %v", err)
	}

	out, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		t.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the remediator can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the namespace, since its `configsync.gke.io/resource-id`
	// annotation is incorrect.
	err = nt.Validate("test-ns", "", &corev1.Namespace{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestKubectlCreatesManagedConfigMapResource(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	nt.Root.Add("acme/cm.yaml", fake.ConfigMapObject(core.Name("cm-1"), core.Namespace("bookstore")))
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	cm := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: bookstore
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _configmap_bookstore_test-cm
    configsync.gke.io/manager: :root
data:
  weekday: "monday"
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-cm.yaml"), cm, 0644); err != nil {
		t.Fatalf("failed to create a tmp file %v", err)
	}

	out, err := nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-cm.yaml"))
	if err != nil {
		t.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
	}

	// Config Sync should remove `test-ns`.
	nomostest.WaitToTerminate(nt, kinds.ConfigMap(), "test-cm", "bookstore")

	cm = []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: bookstore
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _configmap_bookstore_wrong-cm
data:
  weekday: "monday"
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-cm.yaml"), cm, 0644); err != nil {
		t.Fatalf("failed to create a tmp file %v", err)
	}

	out, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-cm.yaml"))
	if err != nil {
		t.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the reconciler can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the configmap, since its `configsync.gke.io/resource-id`
	// annotation is incorrect.
	err = nt.Validate("test-cm", "bookstore", &corev1.ConfigMap{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		t.Fatal(err)
	}
}
