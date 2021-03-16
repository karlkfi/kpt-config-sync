package e2e

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/webhook/configuration"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
)

func TestMustRemoveCustomResourceWithDefinition(t *testing.T) {
	nt := nomostest.New(t)

	nt.Root.Add("acme/cluster/anvil-crd.yaml", anvilV1CRD())
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.Add("acme/namespaces/foo/anvil-v1.yaml", anvilCR("v1", "heavy", 10))
	nt.Root.CommitAndPush("Adding Anvil CRD and one Anvil CR")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	if nt.MultiRepo {
		err := nt.Validate(configuration.Name, "", &admissionv1.ValidatingWebhookConfiguration{},
			hasRule("acme.com.v1.admission-webhook.configsync.gke.io"))
		if err != nil {
			t.Fatal(err)
		}
	}

	err := nt.Validate("heavy", "foo", anvilCR("v1", "", 0))
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err = nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 3,
			metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("CustomResourceDefinition"), metrics.ResourceCreated("Anvil"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// This should cause an error.
	nt.Root.Remove("acme/cluster/anvil-crd.yaml")
	nt.Root.CommitAndPush("Removing Anvil CRD but leaving Anvil CR")

	if nt.MultiRepo {
		nt.WaitForRootSyncSourceError(nonhierarchical.UnsupportedCRDRemovalErrorCode)
	} else {
		nt.WaitForRepoImportErrorCode(nonhierarchical.UnsupportedCRDRemovalErrorCode)
	}

	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		// Validate parse error metric is emitted.
		err = nt.ValidateParseErrors(reconciler.RootSyncName, nonhierarchical.UnsupportedCRDRemovalErrorCode)
		if err != nil {
			return err
		}
		// Validate reconciler error metric is emitted.
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "source")
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// This should fix the error.
	nt.Root.Remove("acme/namespaces/foo/anvil-v1.yaml")
	nt.Root.CommitAndPush("Removing the Anvil CR as well")
	nt.WaitForRepoSyncs()

	// Validate reconciler error is cleared.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		return nt.ValidateReconcilerErrors(reconciler.RootSyncName, "")
	})
	if err != nil {
		t.Errorf("validating reconciler_errors metric: %v", err)
	}
}

func TestLargeCRD(t *testing.T) {
	nt := nomostest.New(t)

	for _, file := range []string{"challenges-acme-cert-manager-io.yaml", "solrclouds-solr-apache-org.yaml"} {
		crdFile := filepath.Join(".", "..", "testdata", "customresources", file)
		crdContent, err := ioutil.ReadFile(crdFile)
		if err != nil {
			t.Fatal(err)
		}
		nt.Root.AddFile(fmt.Sprintf("acme/cluster/%s", file), crdContent)
	}
	nt.Root.CommitAndPush("Adding two large CRDs")
	nt.WaitForRepoSyncs()
	nt.RenewClient()

	err := nt.Validate("challenges.acme.cert-manager.io", "", fake.CustomResourceDefinitionV1Object())
	if err != nil {
		t.Fatal(err)
	}
	err = nt.Validate("solrclouds.solr.apache.org", "", fake.CustomResourceDefinitionV1Object())
	if err != nil {
		t.Fatal(err)
	}

	// Validate multi-repo metrics.
	err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
		nt.ParseMetrics(prev)
		err = nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			metrics.ResourceCreated("CustomResourceDefinition"),
			metrics.ResourceCreated("CustomResourceDefinition"))
		if err != nil {
			return err
		}
		// Validate no error metrics are emitted.
		return nt.ValidateErrorMetricsNotFound()
	})
	if err != nil {
		t.Errorf("validating metrics: %v", err)
	}

	// update one CRD
	crdFile := filepath.Join(".", "..", "testdata", "customresources", "challenges-acme-cert-manager-io_with_new_label.yaml")
	crdContent, err := ioutil.ReadFile(crdFile)
	if err != nil {
		t.Fatal(err)
	}
	nt.Root.AddFile("acme/cluster/challenges-acme-cert-manager-io.yaml", crdContent)
	nt.Root.CommitAndPush("Update label for one CRD")
	nt.WaitForRepoSyncs()

	err = nt.Validate("challenges.acme.cert-manager.io", "", fake.CustomResourceDefinitionV1Beta1Object(), nomostest.HasLabel("random-key", "random-value"))
	if err != nil {
		t.Fatal(err)
	}
}

func hasRule(name string) nomostest.Predicate {
	return func(o core.Object) error {
		vwc, ok := o.(*admissionv1.ValidatingWebhookConfiguration)
		if !ok {
			return nomostest.WrongTypeErr(o, &admissionv1.ValidatingWebhookConfiguration{})
		}
		for _, w := range vwc.Webhooks {
			if w.Name == name {
				return nil
			}
		}
		return errors.Errorf("missing ValidatingWebhook %q", name)
	}
}
