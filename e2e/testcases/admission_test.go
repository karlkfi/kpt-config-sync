package e2e

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
)

// This file includes tests for drift correction and drift prevention.
//
// The drift prevention is only supported in the multi-repo mode, and utilizes the following Config Sync metadata:
//  * the configmanagement.gke.io/managed annotation
//  * the configsync.gke.io/resource-id annotation
//  * the configsync.gke.io/delcared-version label

func TestAdmission(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	nt.RootRepos[configsync.RootSyncName].Add("acme/namespaces/hello/ns.yaml",
		fake.NamespaceObject("hello",
			core.Annotation("goodbye", "moon")))
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add Namespace")
	nt.WaitForRepoSyncs()

	// Ensure we properly forbid changing declared information.

	nomostest.WaitForWebhookReadiness(nt)

	// Prevent deleting declared objects.
	_, err := nt.Kubectl("delete", "ns", "hello")
	if err == nil {
		nt.T.Fatal("got `kubectl delete ns hello` success, want return err")
	}

	// Prevent changing declared data.
	_, err = nt.Kubectl("annotate", "--overwrite", "ns", "hello", "goodbye=world")
	if err == nil {
		nt.T.Fatal("got `kubectl annotate --overwrite ns hello goodbye=world` success, want return err")
	}

	// Prevent removing declared data from declared objects.
	_, err = nt.Kubectl("annotate", "ns", "hello", "goodbye-")
	if err == nil {
		nt.T.Fatal("got `kubectl annotate ns hello goodbye-` success, want return err")
	}

	// Ensure we allow changing information which is not declared.

	// Allow adding data in declared objects.
	out, err := nt.Kubectl("annotate", "ns", "hello", "stop=go")
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate ns hello stop=go` error %v %s, want return nil", err, out)
	}

	// Allow changing non-declared data in declared objects.
	out, err = nt.Kubectl("annotate", "--overwrite", "ns", "hello", "stop='oh no'")
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate --overwrite ns hello stop='oh no'` error %v %s, want return nil", err, out)
	}

	// Allow reing non-declared data in declared objects.
	out, err = nt.Kubectl("annotate", "ns", "hello", "stop-")
	if err != nil {
		nt.T.Fatalf("got `kubectl annotate ns hello stop-` error %v %s, want return nil", err, out)
	}

	// Prevent creating a managed resource.
	ns := []byte(`
apiVersion: v1
kind: Namespace
metadata:
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _namespace_test-ns
  labels:
    configsync.gke.io/declared-version: v1
  name: test-ns
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-ns.yaml"), ns, 0644); err != nil {
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	_, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err == nil {
		nt.T.Fatal("got `kubectl apply -f test-ns.yaml` success, want return err")
	}

	// Allow creating/deleting a resource whose `configsync.gke.io/resource-id` does not match the resource,
	// but whose `configmanagement.gke.io/managed` annotation is `enabled` and whose
	// `configsync.gke.io/declared-version` label is `v1`.
	//
	// The remediator will not remove the Nomos metadata from `test-ns`, since `test-ns` is
	// not a managed resource.
	ns = []byte(`
apiVersion: v1
kind: Namespace
metadata:
  annotations:
    configmanagement.gke.io/managed: enabled
    configsync.gke.io/resource-id: _namespace_wrong-ns
  labels:
    configsync.gke.io/declared-version: v1
  name: test-ns
`)

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "test-ns.yaml"), ns, 0644); err != nil {
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	out, err = nt.Kubectl("apply", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl apply -f test-ns.yaml` error %v %s, want return nil", err, out)
	}

	out, err = nt.Kubectl("delete", "-f", filepath.Join(nt.TmpDir, "test-ns.yaml"))
	if err != nil {
		nt.T.Fatalf("got `kubectl delete -f test-ns.yaml` error %v %s, want return nil", err, out)
	}
}

func TestDisableWebhookConfigurationUpdate(t *testing.T) {
	webhook := []byte(`
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  annotations:
    configsync.gke.io/webhook-configuration-update: disabled
  name: admission-webhook.configsync.gke.io
  labels:
    configmanagement.gke.io/system: "true"
    configmanagement.gke.io/arch: "csmr"
`)

	nt := nomostest.New(t, ntopts.SkipMonoRepo)

	nt.RootRepos[configsync.RootSyncName].Add("acme/namespaces/hello/ns.yaml", fake.NamespaceObject("hello"))
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add Namespace")
	nt.WaitForRepoSyncs()

	if err := ioutil.WriteFile(filepath.Join(nt.TmpDir, "webhook.yaml"), webhook, 0644); err != nil {
		nt.T.Fatalf("failed to create a tmp file %v", err)
	}

	// Recreate the admission webhook
	if _, err := nt.Kubectl("replace", "-f", filepath.Join(nt.TmpDir, "webhook.yaml")); err != nil {
		nt.T.Fatalf("failed to replace the admission webhook %v", err)
	}

	// Verify that the webhook is disabled.
	if _, err := nt.Kubectl("delete", "ns", "hello"); err != nil {
		nt.T.Fatalf("failed to run `kubectl delete ns hello` %v", err)
	}

	// Remove the annotation for disabling webhook
	if _, err := nt.Kubectl("annotate", "ValidatingWebhookConfiguration", "admission-webhook.configsync.gke.io", "configsync.gke.io/webhook-configuration-update-"); err != nil {
		nt.T.Fatalf("failed to remove the annotation in the admission webhook %v", err)
	}

	nt.RootRepos[configsync.RootSyncName].Add("acme/namespaces/test/ns.yaml", fake.NamespaceObject("test"))
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add another Namespace")
	nt.WaitForRepoSyncs()

	nomostest.WaitForWebhookReadiness(nt)

	// Verify that the webhook is now enabled
	if _, err := nt.Kubectl("delete", "ns", "test"); err == nil {
		nt.T.Fatal("got `kubectl delete ns hello` success, want return err")
	}
}
