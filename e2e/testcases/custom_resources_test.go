package e2e

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/discovery"
	"github.com/google/nomos/pkg/webhook/configuration"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCRDDeleteBeforeRemoveCustomResourceV1Beta1(t *testing.T) {
	nt := nomostest.New(t)
	crdFile := filepath.Join(".", "..", "testdata", "customresources", "v1beta1_crds", "anvil-crd.yaml")
	clusterFile := filepath.Join(".", "..", "testdata", "customresources", "v1beta1_crds", "clusteranvil-crd.yaml")
	_, err := nt.Kubectl("apply", "-f", crdFile)
	if err != nil {
		t.Fatal(err)
	}
	_, err = nt.Kubectl("apply", "-f", clusterFile)
	if err != nil {
		t.Fatal(err)
	}
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object(), nomostest.IsEstablished)
	})
	if err != nil {
		t.Fatal(err)
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
			t.Fatal(err)
		}
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate("heavy", "prod", anvilCR("v1", "", 0))
	})
	if err != nil {
		t.Fatal(err)
	}

	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("Anvil"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// Remove CRD
	_, err = nt.Kubectl("delete", "-f", crdFile)
	if err != nil {
		t.Fatal(err)
	}

	if nt.MultiRepo {
		nt.Root.Add("acme/namespaces/prod/anvil-v1.yaml", anvilCR("v1", "heavy", 100))
		nt.Root.CommitAndPush("Adding Anvil CR")
		nt.WaitForRootSyncSourceError(discovery.UnknownKindErrorCode)
	} else {
		nt.WaitForRepoImportErrorCode(discovery.UnknownKindErrorCode)
	}

	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		// Validate parse error metric is emitted.
		err = nt.ValidateParseErrors(reconciler.RootSyncName, discovery.UnknownKindErrorCode)
		if err != nil {
			return err
		}
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
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
		t.Fatal(err)
	}
	_, err = nt.Kubectl("apply", "-f", clusterFile)
	if err != nil {
		t.Fatal(err)
	}
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object(), nomostest.IsEstablished)
	})
	if err != nil {
		t.Fatal(err)
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
			t.Fatal(err)
		}
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.Validate("heavy", "foo", anvilCR("v1", "", 0))
	})
	if err != nil {
		t.Fatal(err)
	}

	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("Anvil"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// Remove CRD
	_, err = nt.Kubectl("delete", "-f", crdFile)
	if err != nil {
		t.Fatal(err)
	}

	if nt.MultiRepo {
		nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 100))
		nt.Root.CommitAndPush("Adding Anvil CR")
		nt.WaitForRootSyncSourceError(discovery.UnknownKindErrorCode)
	} else {
		nt.WaitForRepoImportErrorCode(discovery.UnknownKindErrorCode)
	}

	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		// Validate parse error metric is emitted.
		err = nt.ValidateParseErrors(reconciler.RootSyncName, discovery.UnknownKindErrorCode)
		if err != nil {
			return err
		}
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// Remove the CR.
	// This should fix the error.
	nt.Root.Remove("acme/namespaces/foo/anvil-v1.yaml")
	nt.Root.CommitAndPush("Removing the Anvil CR as well")
	nt.WaitForRepoSyncs()
}

func TestSyncUpdateCustomResource(t *testing.T) {
	nt := nomostest.New(t)
	for _, dir := range []string{"v1beta1_crds"} {
		t.Run(dir, func(t *testing.T) {
			crdFile := filepath.Join(".", "..", "testdata", "customresources", dir, "anvil-crd-structural.yaml")
			_, err := nt.Kubectl("apply", "-f", crdFile)
			if err != nil {
				t.Fatal(err)
			}

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object(), nomostest.IsEstablished)
			})
			if err != nil {
				t.Fatal(err)
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
				t.Fatal(err)
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
				t.Fatal(err)
			}
		})
	}
}

func weightEqual100(obj core.Object) error {
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

func weightEqual10(obj core.Object) error {
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
