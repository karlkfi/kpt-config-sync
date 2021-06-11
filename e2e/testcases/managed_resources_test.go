package e2e

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/webhook/configuration"
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
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	out, err := nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl apply -f test-ns.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the reconciler can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the namespace.
	err = nt.Validate("test-ns", "", &corev1.Namespace{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		nt.T.Fatal(err)
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
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	out, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the reconciler can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the namespace, since its `configsync.gke.io/resource-id`
	// annotation is incorrect.
	err = nt.Validate("test-ns", "", &corev1.Namespace{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		nt.T.Fatal(err)
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
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	out, err := nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl apply -f test-ns.yaml` error %v %s, want return nil", err, out)
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
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	out, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the remediator can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the namespace, since its `configsync.gke.io/resource-id`
	// annotation is incorrect.
	err = nt.Validate("test-ns", "", &corev1.Namespace{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		nt.T.Fatal(err)
	}
}

func TestKubectlCreatesManagedConfigMapResource(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	nt.Root.Add("acme/cm.yaml", fake.ConfigMapObject(core.Name("cm-1"), core.Namespace("bookstore")))
	nt.Root.CommitAndPush("add a configmap")
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
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	out, err := nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-cm.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
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
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	out, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-cm.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl apply -f test-cm.yaml` error %v %s, want return nil", err, out)
	}

	// Wait 10 seconds so that the reconciler can process the event.
	time.Sleep(10 * time.Second)

	// Config Sync should not modify the configmap, since its `configsync.gke.io/resource-id`
	// annotation is incorrect.
	err = nt.Validate("test-cm", "bookstore", &corev1.ConfigMap{}, nomostest.HasExactlyAnnotationKeys(
		v1.ResourceManagementKey, v1beta1.ResourceIDKey, "kubectl.kubernetes.io/last-applied-configuration"))
	if err != nil {
		nt.T.Fatal(err)
	}
}

// TestDeleteManagedResources deletes an object managed by Config Sync,
// and verifies that Config Sync recreates the deleted object.
func TestDeleteManagedResources(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	nt.Root.Add("acme/cm.yaml", fake.ConfigMapObject(core.Name("cm-1"), core.Namespace("bookstore")))
	nt.Root.CommitAndPush("add a configmap")
	nt.WaitForRepoSyncs()

	if nt.MultiRepo {
		// At this point, the Config Sync webhook is on, and should prevent kubectl from deleting a resource managed by Config Sync.
		_, err := nt.Kubectl("delete", "configmap", "cm-1", "-n", "bookstore")
		if err == nil {
			nt.T.Fatalf("got `kubectl delete configmap cm-1` successs, want err")
		}

		_, err = nt.Kubectl("delete", "ns", "bookstore")
		if err == nil {
			nt.T.Fatalf("got `kubectl delete ns bookstore` success, want err")
		}

		stopWebhook(nt)
	}

	// Delete the configmap
	out, err := nt.Kubectl("delete", "configmap", "cm-1", "-n", "bookstore")
	if err != nil {
		nt.T.Fatalf("got `kubectl delete configmap cm-1` error %v %s, want return nil", err, out)
	}

	// Verify Config Sync recreates the configmap
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("cm-1", "bookstore", &corev1.ConfigMap{})
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Delete the namespace
	out, err = nt.Kubectl("delete", "ns", "bookstore")
	if err != nil {
		nt.T.Fatalf("got `kubectl delete ns bookstore` error %v %s, want return nil", err, out)
	}

	// Verify Config Sync recreates the namespace
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("bookstore", "", &corev1.Namespace{})
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

// TestAddFieldsIntoManagedResources adds a new field with kubectl into a resource
// managed by Config Sync, and verifies that Config Sync does not remove this field.
func TestAddFieldsIntoManagedResources(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	// Add a new annotation into the namespace object
	out, err := nt.Kubectl("annotate", "namespace", "bookstore", "season=summer")
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate namespace bookstore season=summer` error %v %s, want return nil", err, out)
	}

	// Verify Config Sync does not remove this field
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("bookstore", "", &corev1.Namespace{}, nomostest.HasAnnotation("season", "summer"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

// TestModifyManagedFields modifies a managed field, and verifies that Config Sync corrects it.
func TestModifyManagedFields(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore", core.Annotation("season", "summer"))
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	if nt.MultiRepo {
		// At this point, the Config Sync webhook is on, and should prevent kubectl from modifying a managed field.
		_, err := nt.Kubectl("annotate", "namespace", "bookstore", "--overwrite", "season=winter")
		if err == nil {
			nt.T.Fatalf("got `kubectl annotate namespace bookstore --overrite season=winter` success, want err")
		}

		// At this point, the Config Sync webhook is on, and should prevent kubectl from modifying Config Sync metadata.
		_, err = nt.Kubectl("annotate", "namespace", "bookstore", "--overwrite", fmt.Sprintf("%s=winter", v1.ResourceManagementKey))
		if err == nil {
			nt.T.Fatalf("got `kubectl annotate namespace bookstore --overwrite %s=winter` success, want err", v1.ResourceManagementKey)
		}
		stopWebhook(nt)
	}

	// Modify a managed field
	out, err := nt.Kubectl("annotate", "namespace", "bookstore", "--overwrite", "season=winter")
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate namespace bookstore --overrite season=winter` error %v %s, want return nil", err, out)
	}

	// Verify Config Sync corrects it
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("bookstore", "", &corev1.Namespace{}, nomostest.HasAnnotation("season", "summer"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Modify a Config Sync annotation
	out, err = nt.Kubectl("annotate", "namespace", "bookstore", "--overwrite", fmt.Sprintf("%s=winter", v1.ResourceManagementKey))
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate namespace bookstore --overwrite %s=winter` error %v %s, want return nil", v1.ResourceManagementKey, err, out)
	}

	// Verify Config Sync corrects it
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("bookstore", "", &corev1.Namespace{}, nomostest.HasAnnotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled))
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

// TestDeleteManagedFields deletes a managed field, and verifies that Config Sync corrects it.
func TestDeleteManagedFields(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore", core.Annotation("season", "summer"))
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush("add a namespace")
	nt.WaitForRepoSyncs()

	if nt.MultiRepo {

		// At this point, the Config Sync webhook is on, and should prevent kubectl from deleting a managed field.
		_, err := nt.Kubectl("annotate", "namespace", "bookstore", "season-")
		if err == nil {
			nt.T.Fatalf("got `kubectl annotate namespace bookstore season-` success, want err")
		}

		// At this point, the Config Sync webhook is on, and should prevent kubectl from deleting Config Sync metadata.
		_, err = nt.Kubectl("annotate", "namespace", "bookstore", fmt.Sprintf("%s-", v1.ResourceManagementKey))
		if err == nil {
			nt.T.Fatalf("got `kubectl annotate namespace bookstore %s-` success, want err", v1.ResourceManagementKey)
		}
		stopWebhook(nt)
	}

	// Delete a managed field
	out, err := nt.Kubectl("annotate", "namespace", "bookstore", "season-")
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate namespace bookstore season-` error %v %s, want return nil", err, out)
	}

	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("bookstore", "", &corev1.Namespace{}, nomostest.HasAnnotation("season", "summer"))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Modify a Config Sync annotation
	out, err = nt.Kubectl("annotate", "namespace", "bookstore", fmt.Sprintf("%s-", v1.ResourceManagementKey))
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate namespace bookstore %s-` error %v %s, want return nil", v1.ResourceManagementKey, err, out)
	}

	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("bookstore", "", &corev1.Namespace{}, nomostest.HasAnnotation(v1.ResourceManagementKey, v1.ResourceManagementEnabled))
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func stopWebhook(nt *nomostest.NT) {
	webhookName := configuration.Name
	webhookGK := "validatingwebhookconfigurations.admissionregistration.k8s.io"

	out, err := nt.Kubectl("annotate", webhookGK, webhookName, fmt.Sprintf("%s=%s", configuration.WebhookconfigurationKey, configuration.WebhookConfigurationUpdateDisabled))
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate %s %s %s=%s` error %v %s, want return nil",
			webhookGK, webhookName, configuration.WebhookconfigurationKey, configuration.WebhookConfigurationUpdateDisabled, err, out)
	}

	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(webhookName, "", &admissionv1.ValidatingWebhookConfiguration{},
			nomostest.HasAnnotation(configuration.WebhookconfigurationKey, configuration.WebhookConfigurationUpdateDisabled))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	out, err = nt.Kubectl("delete", webhookGK, webhookName)
	if err != nil {
		nt.T.Fatalf("got `kubectl delete %s %s` error %v %s, want return nil", webhookGK, webhookName, err, out)
	}

	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.ValidateNotFound(webhookName, "", &admissionv1.ValidatingWebhookConfiguration{})
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
