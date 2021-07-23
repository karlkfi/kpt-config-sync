package e2e

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/testing/fake"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestChangeCustomResourceDefinitionSchema(t *testing.T) {
	nt := nomostest.New(t)

	oldCRDFile := filepath.Join(".", "..", "testdata", "customresources", "changed_schema_crds", "old_schema_crd.yaml")
	newCRDFile := filepath.Join(".", "..", "testdata", "customresources", "changed_schema_crds", "new_schema_crd.yaml")
	newCRFile := filepath.Join(".", "..", "testdata", "customresources", "changed_schema_crds", "new_schema_cr.yaml")

	// Apply a CRD with an old schema.
	nt.MustKubectl("apply", "-f", oldCRDFile)
	_, err := nomostest.Retry(60*time.Second, func() error {
		return nt.Validate("crontabs.stable.example.com", "", &v1.CustomResourceDefinition{}, nomostest.IsEstablished)
	})
	if err != nil {
		nt.T.Fatalf("failed to get the CRD established with old schema")
	}

	// Restart the ConfigSync importer or reconciler pods.
	// So that the old schema of the CRD is picked.
	if nt.MultiRepo {
		nt.MustKubectl("delete", "pods", "-n", "config-management-system", "-l", "configsync.gke.io/reconciler=root-reconciler")
	} else {
		nt.MustKubectl("delete", "pods", "-n", "config-management-system", "-l", "app=git-importer")
	}
	nt.WaitForRepoSyncs()

	// Add the CRD with a new schema to the repo
	crdContent, err := ioutil.ReadFile(newCRDFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.Root.AddFile("acme/cluster/crd.yaml", crdContent)
	nt.Root.CommitAndPush("Adding CRD with new schema")
	nt.WaitForRepoSyncs()

	// Add a CR for the new schema to the repo
	crContent, err := ioutil.ReadFile(newCRFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.AddFile("acme/namespaces/foo/cr.yaml", crContent)
	nt.Root.AddFile("acme/cluster/crd.yaml", crdContent)
	nt.Root.CommitAndPush("Adding namespace and a CR")
	nt.WaitForRepoSyncs()

	err = nt.Validate("my-new-cron-object", "foo", crForNewSchema())
	if err != nil {
		nt.T.Fatal(err)
	}
}

func crForNewSchema() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "stable.example.com",
		Version: "v1",
		Kind:    "CronTab",
	})
	return u
}
