package e2e

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestChangeCustomResourceDefinitionSchema(t *testing.T) {
	nt := nomostest.New(t)

	oldCRDFile := filepath.Join(".", "..", "testdata", "customresources", "changed_schema_crds", "old_schema_crd.yaml")
	newCRDFile := filepath.Join(".", "..", "testdata", "customresources", "changed_schema_crds", "new_schema_crd.yaml")
	oldCRFile := filepath.Join(".", "..", "testdata", "customresources", "changed_schema_crds", "old_schema_cr.yaml")
	newCRFile := filepath.Join(".", "..", "testdata", "customresources", "changed_schema_crds", "new_schema_cr.yaml")

	// Add a CRD and CR to the repo
	crdContent, err := ioutil.ReadFile(oldCRDFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	crContent, err := ioutil.ReadFile(oldCRFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.Root.AddFile("acme/cluster/crd.yaml", crdContent)
	nt.Root.Add("acme/namespaces/foo/ns.yaml", fake.NamespaceObject("foo"))
	nt.Root.AddFile("acme/namespaces/foo/cr.yaml", crContent)
	nt.Root.CommitAndPush("Adding a CRD and CR")
	nt.WaitForRepoSyncs()

	err = nt.Validate("my-cron-object", "foo", crForSchema())
	if err != nil {
		nt.T.Fatal(err)
	}

	// Add the CRD with a new schema and a CR using the new schema to the repo
	crdContent, err = ioutil.ReadFile(newCRDFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	crContent, err = ioutil.ReadFile(newCRFile)
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.Root.AddFile("acme/cluster/crd.yaml", crdContent)
	nt.Root.AddFile("acme/namespaces/foo/cr.yaml", crContent)
	nt.Root.CommitAndPush("Adding the CRD with new schema and a CR using the new schema")
	nt.WaitForRepoSyncs()

	err = nt.Validate("my-new-cron-object", "foo", crForSchema())
	if err != nil {
		nt.T.Fatal(err)
	}
}

func crForSchema() *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "stable.example.com",
		Version: "v1",
		Kind:    "CronTab",
	})
	return u
}
