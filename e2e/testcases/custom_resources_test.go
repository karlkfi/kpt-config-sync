package e2e

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/webhook/configuration"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCRDDeleteBeforeRemoveCustomResourceV1Beta1(t *testing.T) {
	nt := nomostest.New(t)
	support, err := nt.SupportV1Beta1CRD()
	if err != nil {
		nt.T.Fatal("failed to check the supported CRD versions")
	}
	// Skip this test if v1beta1 CRD is not supported in the testing cluster.
	if !support {
		return
	}

	crdFile := filepath.Join(".", "..", "testdata", "customresources", "v1beta1_crds", "anvil-crd.yaml")
	clusterFile := filepath.Join(".", "..", "testdata", "customresources", "v1beta1_crds", "clusteranvil-crd.yaml")
	_, err = nt.Kubectl("apply", "-f", crdFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	_, err = nt.Kubectl("apply", "-f", clusterFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Object(), nomostest.IsEstablished)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.Root.Add("acme/namespaces/prod/ns.yaml", fake.NamespaceObject("prod"))
	nt.Root.Add("acme/namespaces/prod/anvil-v1.yaml", anvilCR("v1", "heavy", 10))
	nt.Root.CommitAndPush("Adding Anvil CR")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	if nt.MultiRepo {
		err := nt.Validate(configuration.Name, "", &admissionv1.ValidatingWebhookConfiguration{},
			hasRule("acme.com.v1.admission-webhook.configsync.gke.io"))
		if err != nil {
			nt.T.Fatal(err)
		}
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate("heavy", "prod", anvilCR("v1", "", 0))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 3,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("Anvil"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): unexpected internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	// Remove CRD
	_, err = nt.Kubectl("delete", "-f", crdFile)
	if err != nil {
		nt.T.Fatal(err)
	}

	if nt.MultiRepo {
		nt.Root.Add("acme/namespaces/prod/anvil-v1.yaml", anvilCR("v1", "heavy", 100))
		nt.Root.CommitAndPush("Adding Anvil CR")
		nt.WaitForRootSyncSourceError(status.UnknownKindErrorCode, "")
	} else {
		nt.WaitForRepoImportErrorCode(status.UnknownKindErrorCode)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		// Validate reconciler error metric is emitted.
		return nt.ReconcilerMetrics.ValidateReconcilerErrors(reconciler.RootSyncName, 1, 1)
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	// Remove the CR.
	// This should fix the error.
	nt.Root.Remove("acme/namespaces/prod/anvil-v1.yaml")
	nt.Root.CommitAndPush("Removing the Anvil CR as well")
	nt.WaitForRepoSyncs()
}

func TestCRDDeleteBeforeRemoveCustomResourceV1(t *testing.T) {
	nt := nomostest.New(t)
	crdFile := filepath.Join(".", "..", "testdata", "customresources", "v1_crds", "anvil-crd.yaml")
	clusterFile := filepath.Join(".", "..", "testdata", "customresources", "v1_crds", "clusteranvil-crd.yaml")
	_, err := nt.Kubectl("apply", "-f", crdFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	_, err = nt.Kubectl("apply", "-f", clusterFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Object(), nomostest.IsEstablished)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 10))
	nt.Root.CommitAndPush("Adding Anvil CR")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	if nt.MultiRepo {
		err := nt.Validate(configuration.Name, "", &admissionv1.ValidatingWebhookConfiguration{},
			hasRule("acme.com.v1.admission-webhook.configsync.gke.io"))
		if err != nil {
			nt.T.Fatal(err)
		}
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate("heavy", "foo", anvilCR("v1", "", 0))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 3,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("Anvil"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		// TODO(b/162601559): unexpected internal_errors_total metric from diff.go
		//return nt.ValidateErrorMetricsNotFound()
		return nil
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	// Remove CRD
	_, err = nt.Kubectl("delete", "-f", crdFile)
	if err != nil {
		nt.T.Fatal(err)
	}

	if nt.MultiRepo {
		nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 100))
		nt.Root.CommitAndPush("Adding Anvil CR")
		nt.WaitForRootSyncSourceError(status.UnknownKindErrorCode, "")
	} else {
		nt.WaitForRepoImportErrorCode(status.UnknownKindErrorCode)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		// Validate reconciler error metric is emitted.
		return nt.ReconcilerMetrics.ValidateReconcilerErrors(reconciler.RootSyncName, 1, 1)
	})
	if err != nil {
		nt.T.Errorf("validating metrics: %v", err)
	}

	// Remove the CR.
	// This should fix the error.
	nt.Root.Remove("acme/namespaces/foo/anvil-v1.yaml")
	nt.Root.CommitAndPush("Removing the Anvil CR as well")
	nt.WaitForRepoSyncs()
}

func TestSyncUpdateCustomResource(t *testing.T) {
	nt := nomostest.New(t)
	support, err := nt.SupportV1Beta1CRD()
	if err != nil {
		nt.T.Fatal("failed to check the supported CRD versions")
	}
	// Skip this test if v1beta1 CRD is not supported in the testing cluster.
	if !support {
		return
	}
	for _, dir := range []string{"v1beta1_crds"} {
		t.Run(dir, func(t *testing.T) {
			crdFile := filepath.Join(".", "..", "testdata", "customresources", dir, "anvil-crd-structural.yaml")
			_, err := nt.Kubectl("apply", "-f", crdFile)
			if err != nil {
				nt.T.Fatal(err)
			}

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Object(), nomostest.IsEstablished)
			})
			if err != nil {
				nt.T.Fatal(err)
			}

			nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
			nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 10))
			nt.Root.CommitAndPush("Adding Anvil CR")
			nt.WaitForRepoSyncs()
			nt.RenewClient()

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("heavy", "foo", anvilCR("v1", "", 0), weightEqual10)
			})
			if err != nil {
				nt.T.Fatal(err)
			}

			// Update CustomResource
			nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 100))
			nt.Root.CommitAndPush("Updating Anvil CR")
			nt.WaitForRepoSyncs()
			nt.RenewClient()

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("heavy", "foo", anvilCR("v1", "", 0), weightEqual100)
			})
			if err != nil {
				nt.T.Fatal(err)
			}
		})
	}
}

func weightEqual100(obj client.Object) error {
	u := obj.(*unstructured.Unstructured)
	val, _, err := unstructured.NestedInt64(u.Object, "spec", "lbs")
	if err != nil {
		return err
	}
	if val != 100 {
		return fmt.Errorf(".spec.lbs should be 100 but got %d", val)
	}
	return nil
}

func weightEqual10(obj client.Object) error {
	u := obj.(*unstructured.Unstructured)
	val, _, err := unstructured.NestedInt64(u.Object, "spec", "lbs")
	if err != nil {
		return err
	}
	if val != 10 {
		return fmt.Errorf(".spec.lbs should be 10 but got %d", val)
	}
	return nil
}
