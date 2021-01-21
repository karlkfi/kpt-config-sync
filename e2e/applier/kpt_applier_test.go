// This e2e test is added to check the correctness of the new kpt applier.
// When we switch over to the kptapplier in the reconciler and all other
// e2e tests pass, we can safely remove this e2e test.

package applier

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var cfg *rest.Config
var k8sClient client.Client
var ctx context.Context
var orgKubeConfigEnv string

func prepare(t *testing.T) {
	var err error

	optsStruct := ntopts.New{}
	ntopts.Kind(t, *e2e.KubernetesVersion)(&optsStruct)
	cfg = optsStruct.RESTConfig

	orgKubeConfigEnv = os.Getenv("KUBECONFIG")
	err = os.Setenv("KUBECONFIG", ntopts.Kubeconfig)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// Apply the CRD
	ctx = context.Background()
	rgCRD := resourcegroupCRD(t)
	err = k8sClient.Create(ctx, rgCRD)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	err = waitForCRD(rgCRD)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	// Create the namespace config-management-system
	namespace := configManagementNamespace()
	err = k8sClient.Create(ctx, namespace)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
}

func teardown(t *testing.T) {
	rgCRD := resourcegroupCRD(t)
	err := k8sClient.Delete(ctx, rgCRD)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	namespace := configManagementNamespace()
	err = k8sClient.Delete(ctx, namespace)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	err = os.Setenv("KUBECONFIG", orgKubeConfigEnv)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
}

func configManagementNamespace() *unstructured.Unstructured {
	namespace := fake.UnstructuredObject(kinds.Namespace(),
		core.Name("config-management-system"))
	return namespace
}

func resourcegroupCRD(t *testing.T) *unstructured.Unstructured {
	data, err := ioutil.ReadFile(filepath.Join("..", "..", "manifests", "test-resources", "kpt-resourcegroup-crd.yaml"))
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	u := &unstructured.Unstructured{}
	err = yaml.Unmarshal(data, u)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	return u
}

func waitForCRD(crd *unstructured.Unstructured) error {
	_, err := nomostest.Retry(120*time.Second, func() error {
		return k8sClient.Get(ctx, client.ObjectKey{
			Name: crd.GetName(),
		}, crd)
	})
	return err
}

func TestApplier(t *testing.T) {
	if !*e2e.E2E {
		return
	}
	prepare(t)
	defer teardown(t)

	testNamespaceApplier(t)
	testRootApplier(t)
	testConflictResource(t)
	testUnknownType(t)
	testDisabledResource(t)
}

const (
	testCM1 = "fake-configmap-1"
	testCM2 = "fake-configmap-2"
	testCM3 = "fake-configmap-3"
)

func testNamespaceApplier(t *testing.T) {
	t.Log("namespace applier: first apply, then prune, then clean up")
	declaredResources := []ast.FileObject{
		fake.ConfigMapAtPath(testCM1, core.Name(testCM1), core.Namespace("default"), syncertest.ManagementEnabled),
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ConfigMap(): {},
	}
	resources := filesystem.AsCoreObjects(declaredResources)
	applier := kptapplier.NewNamespaceApplier(k8sClient, "default")
	gvks, errs := applier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	if diff := cmp.Diff(wantGVKs, gvks); diff != "" {
		t.Errorf("Diff of GVK map from Apply(): %s", diff)
	}

	cmObject := fake.UnstructuredObject(kinds.ConfigMap())
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: "default",
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	declaredResources = []ast.FileObject{
		fake.ConfigMapAtPath(testCM2, core.Name(testCM2), core.Namespace("default"), syncertest.ManagementEnabled),
		fake.ConfigMapAtPath(testCM3, core.Name(testCM3), core.Namespace("default"), syncertest.ManagementEnabled),
	}

	_, errs = applier.Apply(ctx, filesystem.AsCoreObjects(declaredResources))
	if err != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm testCM2, testCM3 are applied and testCM1 is pruned.
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM2,
		Namespace: "default",
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM3,
		Namespace: "default",
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: "default",
	}, cmObject)
	if err == nil {
		t.Errorf("get testCM1 should return an error")
	}

	_, errs = applier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	// confirm testCM2 and testCM3 are pruned
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM2,
		Namespace: "default",
	}, cmObject)
	if err == nil {
		t.Errorf("get testCM2 should return an error")
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM3,
		Namespace: "default",
	}, cmObject)
	if err == nil {
		t.Errorf("get testCM3 should return an error")
	}
}

const (
	testCR1 = "fake-clusterrole-1"
	testCR2 = "fake-clusterrole-2"
	testCR3 = "fake-clusterrole-3"
)

func testRootApplier(t *testing.T) {
	t.Log("root applier: first apply, then prune, then clean up")
	declared := []ast.FileObject{
		fake.ClusterRole(core.Name(testCR1), syncertest.ManagementEnabled),
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ClusterRole(): {},
	}
	resources := filesystem.AsCoreObjects(declared)
	applier := kptapplier.NewRootApplier(k8sClient)
	gvks, errs := applier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	if diff := cmp.Diff(wantGVKs, gvks); diff != "" {
		t.Errorf("Diff of GVK map from Apply(): %s", diff)
	}
	clusterRoleObj := fake.UnstructuredObject(kinds.ClusterRole())
	// confirm testCR1 is applied.
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCR1,
		Namespace: "",
	}, clusterRoleObj)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	declared = []ast.FileObject{
		fake.ClusterRole(core.Name(testCR2), syncertest.ManagementEnabled),
		fake.ClusterRole(core.Name(testCR3), syncertest.ManagementEnabled),
	}
	_, errs = applier.Apply(ctx, filesystem.AsCoreObjects(declared))
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	// confirm testCR2, testCR3 are applied and testCR1 is pruned.
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: testCR2,
	}, clusterRoleObj)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: testCR3,
	}, clusterRoleObj)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: testCR1,
	}, clusterRoleObj)
	if err == nil {
		t.Errorf("get testCR1 should return an error")
	}

	// Apply the empty resource list
	_, errs = applier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm that testCR2 and testCR3 are pruned.
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: testCR2,
	}, clusterRoleObj)
	if err == nil {
		t.Errorf("get testCR2 should return an error")
	}
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: testCR3,
	}, clusterRoleObj)
	if err == nil {
		t.Errorf("get testCR3 should return an error")
	}
}

const testRole = "fake-role"

func testConflictResource(t *testing.T) {
	t.Log("root applier and namespaced applier has conflict")
	nsDeclared := []ast.FileObject{
		fake.Role(core.Name(testRole), core.Namespace("default"),
			core.Annotation("from", "namespace applier"),
			syncertest.ManagementEnabled),
	}
	resources := filesystem.AsCoreObjects(nsDeclared)
	nsApplier := kptapplier.NewNamespaceApplier(k8sClient, "default")
	_, errs := nsApplier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	rootDeclared := []ast.FileObject{
		fake.Role(core.Name(testRole), core.Namespace("default"),
			core.Annotation("from", "root applier"),
			syncertest.ManagementEnabled),
	}
	rootResources := filesystem.AsCoreObjects(rootDeclared)
	rootApplier := kptapplier.NewRootApplier(k8sClient)
	_, errs = rootApplier.Apply(ctx, rootResources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	_, errs = nsApplier.Apply(ctx, resources)
	if errs == nil {
		t.Fatalf("namespace applier should return an error")
	}

	role := fake.UnstructuredObject(kinds.Role(), core.Name(testRole), core.Namespace("default"))
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      testRole,
		Namespace: "default",
	}, role)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Confirm the cluster object is from the root applier.
	if role.GetAnnotations()["from"] != "root applier" {
		t.Errorf("should be applied from the root applier")
	}

	// Remove the resource from the root applier.
	// Reapply both the namespace and root resources.
	_, errs = rootApplier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	_, errs = nsApplier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testRole,
		Namespace: "default",
	}, role)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// confirm that the resource is from the namespace applier.
	if role.GetAnnotations()["from"] != "namespace applier" {
		t.Errorf("should be applied from the namespace applier")
	}

}

func testUnknownType(t *testing.T) {
	t.Log("namespace applier: apply two resources; one has unknown type")
	cr := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "test.io/v1",
			"kind":       "Unknown",
			"metadata": map[string]interface{}{
				"name":      "unknown",
				"namespace": "default",
			},
		},
	}
	declaredResources := []ast.FileObject{
		fake.ConfigMapAtPath(testCM1, core.Name(testCM1), core.Namespace("default"), syncertest.ManagementEnabled),
		fake.FileObject(cr, "cr.yaml"),
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ConfigMap(): {},
	}
	resources := filesystem.AsCoreObjects(declaredResources)
	applier := kptapplier.NewNamespaceApplier(k8sClient, "default")
	gvks, errs := applier.Apply(ctx, resources)
	if errs == nil || !strings.Contains(errs.Error(), "no matches for kind \"Unknown\" in version \"test.io/v1\"") {
		t.Errorf("an unknown type error should happen %v", errs)
	}
	if diff := cmp.Diff(wantGVKs, gvks); diff != "" {
		t.Errorf("Diff of GVK map from Apply(): %s", diff)
	}

	crd, err := getCRD()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	rootResources := []ast.FileObject{
		fake.FileObject(crd, "crd.yaml"),
	}
	rootApplier := kptapplier.NewRootApplier(k8sClient)
	_, errs = rootApplier.Apply(ctx, filesystem.AsCoreObjects(rootResources))
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	// Reapply the namespace applier
	wantGVKs = map[schema.GroupVersionKind]struct{}{
		kinds.ConfigMap(): {},
		{
			Group:   "test.io",
			Version: "v1",
			Kind:    "Unknown",
		}: {},
	}
	gvks, errs = applier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	if diff := cmp.Diff(wantGVKs, gvks); diff != "" {
		t.Errorf("Diff of GVK map from Apply(): %s", diff)
	}
}

func testDisabledResource(t *testing.T) {
	t.Log("namespace applier: first resource apply, then disable the resource.")
	declaredResources := []ast.FileObject{
		fake.ConfigMapAtPath(testCM1, core.Name(testCM1), core.Namespace("default"), syncertest.ManagementEnabled),
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ConfigMap(): {},
	}
	resources := filesystem.AsCoreObjects(declaredResources)
	applier := kptapplier.NewNamespaceApplier(k8sClient, "default")
	gvks, errs := applier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	if diff := cmp.Diff(wantGVKs, gvks); diff != "" {
		t.Errorf("Diff of GVK map from Apply(): %s", diff)
	}

	cmObject := fake.UnstructuredObject(kinds.ConfigMap())
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: "default",
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// disable the object
	declaredResources = []ast.FileObject{
		fake.ConfigMapAtPath(testCM1, core.Name(testCM1), core.Namespace("default"), syncertest.ManagementDisabled),
	}
	_, errs = applier.Apply(ctx, filesystem.AsCoreObjects(declaredResources))
	if err != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm testCM1 exists and doesn't contain nomos metadata.
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: "default",
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if _, found := cmObject.GetLabels()[v1.ManagedByKey]; found {
		t.Errorf("should have removed the label %s", v1.ManagedByKey)
	}
	if _, found := cmObject.GetAnnotations()["config.k8s.io/owning-inventory"]; found {
		t.Errorf("should have removed the annotation %s", "config.k8s.io/owning-inventory")
	}

	t.Log("namespace applier: applier doesn't prune the disabled resource")
	_, errs = applier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm testCM1 exists
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: "default",
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	t.Log("namespace applier: applier can manage the disabled resource again when enabling")
	declaredResources = []ast.FileObject{
		fake.ConfigMapAtPath(testCM1, core.Name(testCM1), core.Namespace("default"), syncertest.ManagementEnabled),
	}
	_, errs = applier.Apply(ctx, filesystem.AsCoreObjects(declaredResources))
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm testCM1 exists
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: "default",
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if _, found := cmObject.GetLabels()[v1.ManagedByKey]; !found {
		t.Errorf("should have the label %s", v1.ManagedByKey)
	}
	if _, found := cmObject.GetAnnotations()["config.k8s.io/owning-inventory"]; !found {
		t.Errorf("should have the annotation %s", "config.k8s.io/owning-inventory")
	}
}

func getCRD() (*unstructured.Unstructured, error) {
	crdManifest := `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: unknowns.test.io
spec:
  group: test.io
  versions:
    - name: v1
      served: true
      storage: true
  scope: Namespaced
  names:
    plural: unknowns
    singular: unknown
    kind: Unknown
    shortNames:
    - un
`

	crd := &unstructured.Unstructured{}
	err := yaml.Unmarshal([]byte(crdManifest), crd)
	return crd, err
}
