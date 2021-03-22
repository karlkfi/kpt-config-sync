package e2e

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/metrics"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/webhook/configuration"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMustRemoveCustomResourceWithDefinition(t *testing.T) {
	nt := nomostest.New(t)
	testcases := []struct {
		name string
		fn   func() client.Object
	}{
		{
			name: "v1 crd",
			fn:   func() client.Object { return anvilV1CRD() },
		},
		{
			name: "v1beta1 crd",
			fn:   func() client.Object { return anvilV1Beta1CRD() },
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			nt.Root.Add("acme/cluster/anvil-crd.yaml", tc.fn())
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
				return nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 3,
					metrics.ResourceCreated("Namespace"), metrics.ResourceCreated("CustomResourceDefinition"), metrics.ResourceCreated("Anvil"))
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
		})
	}
}

func TestAddAndRemoveCustomResource(t *testing.T) {
	nt := nomostest.New(t)

	for _, dir := range []string{"v1_crds", "v1beta1_crds"} {
		t.Run(dir, func(t *testing.T) {
			crdFile := filepath.Join(".", "..", "testdata", "customresources", dir, "anvil-crd.yaml")
			crdContent, err := ioutil.ReadFile(crdFile)
			if err != nil {
				t.Fatal(err)
			}
			nt.Root.AddFile("acme/cluster/anvil-crd.yaml", crdContent)
			nt.Root.Add("acme/namespaces/prod/ns.yaml", fake.NamespaceObject("prod"))
			nt.Root.Add("acme/namespaces/prod/anvil.yaml", anvilCR("v1", "e2e-test-anvil", 10))
			nt.Root.CommitAndPush("Adding Anvil CRD and one Anvil CR")
			nt.WaitForRepoSyncs()
			nt.RenewClient()

			err = nt.Validate("e2e-test-anvil", "prod", anvilCR("v1", "", 10))
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
				return nil
			})
			if err != nil {
				t.Errorf("validating metrics: %v", err)
			}

			// Remove the CustomResource.
			nt.Root.Remove("acme/namespaces/prod/anvil.yaml")
			nt.Root.CommitAndPush("Removing Anvil CR but leaving Anvil CRD")
			nt.WaitForRepoSyncs()
			err = nt.ValidateNotFound("e2e-test-anvil", "prod", anvilCR("v1", "", 10))
			if err != nil {
				t.Fatal(err)
			}

			// Remove the CustomResourceDefinition.
			nt.Root.Remove("acme/cluster/anvil-crd.yaml")
			nt.Root.CommitAndPush("Removing the Anvil CRD as well")
			nt.WaitForRepoSyncs()
			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.ValidateNotFound("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object())
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMustRemoveUnManagedCustomResource(t *testing.T) {
	nt := nomostest.New(t)

	for _, dir := range []string{"v1_crds", "v1beta1_crds"} {
		t.Run(dir, func(t *testing.T) {
			crdFile := filepath.Join(".", "..", "testdata", "customresources", dir, "anvil-crd.yaml")
			crdContent, err := ioutil.ReadFile(crdFile)
			if err != nil {
				t.Fatal(err)
			}
			nt.Root.AddFile("acme/cluster/anvil-crd.yaml", crdContent)
			nt.Root.Add("acme/namespaces/prod/ns.yaml", fake.NamespaceObject("prod"))
			nt.Root.CommitAndPush("Adding Anvil CRD")
			nt.WaitForRepoSyncs()
			nt.RenewClient()

			// TODO: Fix the multi-repo metrics error.
			// Validate multi-repo metrics.
			//err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
			//	nt.ParseMetrics(prev)
			//	err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
			//		metrics.ResourceCreated("CustomResourceDefinition"),
			//		metrics.ResourceCreated("Namespace"))
			//	return err
			//})
			//if err != nil {
			//	t.Errorf("validating metrics: %v", err)
			//}

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object())
			})
			if err != nil {
				t.Fatal(err)
			}

			// Apply the CustomResource.
			cr := anvilCR("v1", "e2e-test-anvil", 100)
			cr.SetNamespace("prod")
			err = nt.Client.Create(context.TODO(), cr)
			if err != nil {
				t.Fatal(err)
			}

			// Remove the CustomResourceDefinition.
			nt.Root.Remove("acme/cluster/anvil-crd.yaml")
			nt.Root.CommitAndPush("Removing the Anvil CRD")
			nt.WaitForRepoSyncs()

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.ValidateNotFound("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object())
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAddUpdateClusterScopedCRD(t *testing.T) {
	nt := nomostest.New(t)
	for _, dir := range []string{"v1_crds", "v1beta1_crds"} {
		t.Run(dir, func(t *testing.T) {
			crdFile := filepath.Join(".", "..", "testdata", "customresources", dir, "clusteranvil-crd.yaml")
			crdContent, err := ioutil.ReadFile(crdFile)
			if err != nil {
				t.Fatal(err)
			}
			nt.Root.AddFile("acme/cluster/clusteranvil-crd.yaml", crdContent)
			nt.Root.Add("acme/cluster/clusteranvil.yaml", clusteranvilCR("v1", "e2e-test-clusteranvil", 10))
			nt.Root.CommitAndPush("Adding clusterscoped Anvil CRD and CR")
			nt.WaitForRepoSyncs()
			nt.RenewClient()

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("e2e-test-clusteranvil", "", clusteranvilCR("v1", "", 10))
			})
			if err != nil {
				t.Fatal(err)
			}

			// Validate multi-repo metrics.
			err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
				nt.ParseMetrics(prev)
				err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 2,
					metrics.ResourceCreated("CustomResourceDefinition"),
					metrics.ResourceCreated("ClusterAnvil"))
				if err != nil {
					return err
				}
				// Validate no error metrics are emitted.
				return nt.ValidateErrorMetricsNotFound()
			})
			if err != nil {
				t.Errorf("validating metrics: %v", err)
			}

			// Update the CRD from version v1 to version v2.
			crdFile = filepath.Join(".", "..", "testdata", "customresources", dir, "clusteranvil-crd-v2.yaml")
			crdContent, err = ioutil.ReadFile(crdFile)
			if err != nil {
				t.Fatal(err)
			}
			nt.Root.AddFile("acme/cluster/clusteranvil-crd.yaml", crdContent)
			nt.Root.CommitAndPush("Updating the Anvil CRD")
			nt.WaitForRepoSyncs()

			err = nt.Validate("clusteranvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object(), hasTwoVersions)
			if err != nil {
				t.Fatal(err)
			}
			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("e2e-test-clusteranvil", "", clusteranvilCR("v2", "", 10))
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAddUpdateNamespaceScopedCRD(t *testing.T) {
	nt := nomostest.New(t)

	for _, dir := range []string{"v1_crds", "v1beta1_crds"} {
		t.Run(dir, func(t *testing.T) {
			crdFile := filepath.Join(".", "..", "testdata", "customresources", dir, "anvil-crd.yaml")
			crdContent, err := ioutil.ReadFile(crdFile)
			if err != nil {
				t.Fatal(err)
			}
			nt.Root.AddFile("acme/cluster/anvil-crd.yaml", crdContent)
			nt.Root.Add("acme/namespaces/prod/anvil.yaml", anvilCR("v1", "e2e-test-anvil", 10))
			nt.Root.Add("acme/namespaces/prod/ns.yaml", fake.NamespaceObject("prod"))
			nt.Root.CommitAndPush("Adding namespacescoped Anvil CRD and CR")
			nt.WaitForRepoSyncs()
			nt.RenewClient()

			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.Validate("e2e-test-anvil", "prod", anvilCR("v1", "", 10))
			})
			if err != nil {
				t.Fatal(err)
			}

			// Validate multi-repo metrics.
			err = nt.RetryMetrics(60*time.Second, func(prev metrics.ConfigSyncMetrics) error {
				nt.ParseMetrics(prev)
				err := nt.ValidateMultiRepoMetrics(reconciler.RootSyncName, 3,
					metrics.ResourceCreated("CustomResourceDefinition"),
					metrics.ResourceCreated("Anvil"),
					metrics.ResourceCreated("Namespace"))
				return err
			})
			if err != nil {
				t.Errorf("validating metrics: %v", err)
			}

			// Update the CRD from version v1 to version v2.
			crdFile = filepath.Join(".", "..", "testdata", "customresources", dir, "anvil-crd-v2.yaml")
			crdContent, err = ioutil.ReadFile(crdFile)
			if err != nil {
				t.Fatal(err)
			}
			nt.Root.AddFile("acme/cluster/anvil-crd.yaml", crdContent)
			nt.Root.CommitAndPush("Updating the Anvil CRD")
			nt.WaitForRepoSyncs()

			err = nt.Validate("e2e-test-anvil", "prod", anvilCR("v2", "", 10))
			if err != nil {
				t.Fatal(err)
			}
			err = nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object(), hasTwoVersions)
			if err != nil {
				t.Fatal(err)
			}

			// Update CRD and CR to only support V2
			crdFile = filepath.Join(".", "..", "testdata", "customresources", dir, "anvil-crd-only-v2.yaml")
			crdContent, err = ioutil.ReadFile(crdFile)
			if err != nil {
				t.Fatal(err)
			}
			nt.Root.AddFile("acme/cluster/anvil-crd.yaml", crdContent)
			nt.Root.Add("acme/namespaces/prod/anvil.yaml", anvilCR("v2", "e2e-test-anvil", 10))
			nt.Root.CommitAndPush("Update the Anvil CRD and CR")
			nt.WaitForRepoSyncs()

			_, err = nomostest.Retry(60*time.Second, func() error {
				return nt.Validate("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object(), nomostest.IsEstablished, hasTwoVersions)
			})
			if err != nil {
				t.Fatal(err)
			}

			err = nt.Validate("e2e-test-anvil", "prod", anvilCR("v2", "", 10))
			if err != nil {
				t.Fatal(err)
			}

			// Remove CRD and CR
			nt.Root.Remove("acme/cluster/anvil-crd.yaml")
			nt.Root.Remove("acme/namespaces/prod/anvil.yaml")
			nt.Root.CommitAndPush("Remove the Anvil CRD and CR")
			nt.WaitForRepoSyncs()

			// Validate the CustomResource is also deleted from cluster.
			_, err = nomostest.Retry(30*time.Second, func() error {
				return nt.ValidateNotFound("anvils.acme.com", "", fake.CustomResourceDefinitionV1Beta1Object())
			})
			if err != nil {
				t.Fatal(err)
			}
		})
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
	return func(o client.Object) error {
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

func hasTwoVersions(obj client.Object) error {
	crd := obj.(*apiextensionsv1beta1.CustomResourceDefinition)
	if len(crd.Spec.Versions) != 2 {
		return errors.New("the CRD should contain 2 versions")
	}
	if crd.Spec.Versions[0].Name != "v1" || crd.Spec.Versions[1].Name != "v2" {
		return errors.New("incorrect versions for CRD")
	}
	return nil
}

func clusteranvilCR(version, name string, weight int64) *unstructured.Unstructured {
	u := anvilCR(version, name, weight)
	gvk := u.GroupVersionKind()
	gvk.Kind = "ClusterAnvil"
	u.SetGroupVersionKind(gvk)
	return u
}
