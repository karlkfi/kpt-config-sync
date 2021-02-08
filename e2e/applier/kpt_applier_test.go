// This e2e test is added to check the correctness of the new kpt applier.
// When we switch over to the kptapplier in the reconciler and all other
// e2e tests pass, we can safely remove this e2e test.

package applier

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/e2e"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/kptapplier"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func prepare(t *testing.T) {
	var err error

	optsStruct := ntopts.New{
		TmpDir: nomostest.TestDir(t),
	}
	nomostest.RestConfig(t, &optsStruct)
	cfg = optsStruct.RESTConfig
	if err = os.Setenv(ntopts.Kubeconfig, filepath.Join(optsStruct.TmpDir, ntopts.Kubeconfig)); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	s := scheme.Scheme
	err = v1beta1.AddToScheme(s)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	k8sClient, err = client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if err := cleanup(); err != nil {
		t.Fatalf("failed to clean up before the test %v", err)
	}
	t.Cleanup(func() {
		if err := cleanup(); err != nil {
			t.Logf("failed to clean up after the test %v", err)
		}
		if err := os.Unsetenv(ntopts.Kubeconfig); err != nil {
			t.Log(err)
		}
	})

	// Apply the CRD
	ctx = context.Background()
	rgCRD, err := resourcegroupCRD()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
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

func cleanup() error {
	// Delete the CRD
	ctx = context.Background()
	rgCRD, err := resourcegroupCRD()
	if err != nil {
		return err
	}
	err = k8sClient.Delete(ctx, rgCRD)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Delete the namespace config-management-system
	namespace := configManagementNamespace()
	err = k8sClient.Delete(ctx, namespace)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	_, err = nomostest.Retry(120*time.Second, func() error {
		err2 := k8sClient.Get(ctx, client.ObjectKey{
			Name: namespace.GetName(),
		}, namespace)
		if err2 != nil {
			if apierrors.IsNotFound(err2) {
				return nil
			}
			return err2
		}
		return fmt.Errorf("namespace hasn't been removed")
	})
	return err
}

func configManagementNamespace() *unstructured.Unstructured {
	namespace := fake.UnstructuredObject(kinds.Namespace(),
		core.Name("config-management-system"))
	return namespace
}

func resourcegroupCRD() (*unstructured.Unstructured, error) {
	data, err := ioutil.ReadFile(filepath.Join("..", "..", "manifests", "test-resources", "kpt-resourcegroup-crd.yaml"))
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	err = yaml.Unmarshal(data, u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func waitForCRD(crd *unstructured.Unstructured) error {
	_, err := nomostest.Retry(120*time.Second, func() error {
		obj := &v1beta1.CustomResourceDefinition{}
		err2 := k8sClient.Get(ctx, client.ObjectKey{
			Name: crd.GetName(),
		}, obj)
		if err2 != nil {
			return err2
		}
		return nomostest.IsEstablished(obj)
	})
	return err
}

const (
	testCM1 = "fake-configmap-1"
	testCM2 = "fake-configmap-2"
	testCM3 = "fake-configmap-3"
)

func TestNamespaceApplier(t *testing.T) {
	if !*e2e.E2E || !*e2e.MultiRepo {
		return
	}
	prepare(t)
	t.Log("namespace applier: first apply, then prune, then clean up")
	resources := []core.Object{
		fake.ConfigMapObject(core.Name(testCM1), core.Namespace(metav1.NamespaceDefault), syncertest.ManagementEnabled),
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ConfigMap(): {},
	}

	applier := kptapplier.NewNamespaceApplier(k8sClient, declared.Scope(metav1.NamespaceDefault))
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
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	resources = []core.Object{
		fake.ConfigMapObject(core.Name(testCM2), core.Namespace(metav1.NamespaceDefault), syncertest.ManagementEnabled),
		fake.ConfigMapObject(core.Name(testCM3), core.Namespace(metav1.NamespaceDefault), syncertest.ManagementEnabled),
	}

	_, errs = applier.Apply(ctx, resources)
	if err != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm testCM2, testCM3 are applied and testCM1 is pruned.
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM2,
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM3,
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err == nil {
		t.Errorf("get testCM1 should return an error")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("got Get() = %v, want IsNotFound", err)
	}

	_, errs = applier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	// confirm testCM2 and testCM3 are pruned
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM2,
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err == nil {
		t.Errorf("get testCM2 should return an error")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("got Get() = %v, want IsNotFound", err)
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM3,
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err == nil {
		t.Errorf("get testCM3 should return an error")
	}
	if !apierrors.IsNotFound(err) {
		t.Errorf("got Get() = %v, want IsNotFound", err)
	}
}

const (
	testCR1 = "fake-clusterrole-1"
	testCR2 = "fake-clusterrole-2"
	testCR3 = "fake-clusterrole-3"
)

func TestRootApplier(t *testing.T) {
	if !*e2e.E2E || !*e2e.MultiRepo {
		return
	}
	prepare(t)
	t.Log("root applier: first apply, then prune, then clean up")
	resources := []core.Object{
		fake.ClusterRoleObject(core.Name(testCR1), syncertest.ManagementEnabled),
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ClusterRole(): {},
	}

	applier := kptapplier.NewNamespaceApplier(k8sClient, declared.Scope(metav1.NamespaceDefault))
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

	resources = []core.Object{
		fake.ClusterRoleObject(core.Name(testCR2), syncertest.ManagementEnabled),
		fake.ClusterRoleObject(core.Name(testCR3), syncertest.ManagementEnabled),
	}
	_, errs = applier.Apply(ctx, resources)
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
	if !apierrors.IsNotFound(err) {
		t.Errorf("got Get() = %v, want IsNotFound", err)
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
	if !apierrors.IsNotFound(err) {
		t.Errorf("got Get() = %v, want IsNotFound", err)
	}

	err = k8sClient.Get(ctx, client.ObjectKey{
		Name: testCR3,
	}, clusterRoleObj)
	if err == nil {
		t.Errorf("get testCR3 should return an error")
	}

	if !apierrors.IsNotFound(err) {
		t.Errorf("got Get() = %v, want IsNotFound", err)
	}
}

const testRole = "fake-role"

func TestConflictResource(t *testing.T) {
	if !*e2e.E2E || !*e2e.MultiRepo {
		return
	}
	prepare(t)
	t.Log("root applier and namespaced applier has conflict")
	resources := []core.Object{
		fake.RoleObject(core.Name(testRole), core.Namespace(metav1.NamespaceDefault),
			core.Annotation("from", "namespace applier"),
			syncertest.ManagementEnabled),
	}

	nsApplier := kptapplier.NewNamespaceApplier(k8sClient, declared.Scope(metav1.NamespaceDefault))
	_, errs := nsApplier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	rootResources := []core.Object{
		fake.RoleObject(core.Name(testRole), core.Namespace(metav1.NamespaceDefault),
			core.Annotation("from", "root applier"),
			syncertest.ManagementEnabled),
	}
	rootApplier := kptapplier.NewRootApplier(k8sClient)
	_, errs = rootApplier.Apply(ctx, rootResources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	_, errs = nsApplier.Apply(ctx, resources)
	if errs == nil {
		t.Fatalf("namespace applier should return an error")
	}

	role := fake.UnstructuredObject(kinds.Role(), core.Name(testRole), core.Namespace(metav1.NamespaceDefault))
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      testRole,
		Namespace: metav1.NamespaceDefault,
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
		Namespace: metav1.NamespaceDefault,
	}, role)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// confirm that the resource is from the namespace applier.
	if role.GetAnnotations()["from"] != "namespace applier" {
		t.Errorf("should be applied from the namespace applier")
	}

	// cleanup the applied resources
	_, errs = nsApplier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	_, errs = rootApplier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
}

func TestUnknownType(t *testing.T) {
	if !*e2e.E2E || !*e2e.MultiRepo {
		return
	}
	prepare(t)
	t.Log("namespace applier: apply three resources; two have unknown type")
	cr1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "test.io/v1",
			"kind":       "Unknown",
			"metadata": map[string]interface{}{
				"name":      "unknown",
				"namespace": metav1.NamespaceDefault,
			},
		},
	}
	cr2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "test.io/v1",
			"kind":       "Example",
			"metadata": map[string]interface{}{
				"name":      "example",
				"namespace": metav1.NamespaceDefault,
			},
		},
	}
	resources := []core.Object{
		fake.ConfigMapObject(core.Name(testCM1), core.Namespace(metav1.NamespaceDefault), syncertest.ManagementEnabled),
		cr1,
		cr2,
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ConfigMap(): {},
	}

	applier := kptapplier.NewNamespaceApplier(k8sClient, declared.Scope(metav1.NamespaceDefault))
	gvks, errs := applier.Apply(ctx, resources)
	if errs == nil || !strings.Contains(errs.Error(), "no matches for kind \"Unknown\" in version \"test.io/v1\"") {
		t.Errorf("an unknown type error should happen %v", errs)
	}
	if diff := cmp.Diff(wantGVKs, gvks); diff != "" {
		t.Errorf("Diff of GVK map from Apply(): %s", diff)
	}

	crd1, err := getCRDV1beta1()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	crd2, err := getCRDV1()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	rootResources := []core.Object{
		crd1,
		crd2,
	}
	rootApplier := kptapplier.NewRootApplier(k8sClient)
	_, errs = rootApplier.Apply(ctx, rootResources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}

	err = waitForCRD(crd1)
	if err != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	err = waitForCRD(crd2)
	if err != nil {
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
		{
			Group:   "test.io",
			Version: "v1",
			Kind:    "Example",
		}: {},
	}
	gvks, errs = applier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	if diff := cmp.Diff(wantGVKs, gvks, cmpopts.SortSlices(
		func(x, y schema.GroupVersionKind) bool { return x.String() < y.String() })); diff != "" {
		t.Errorf("Diff of GVK map from Apply(): %s", diff)
	}

	// cleanup the applied resources
	_, errs = applier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	_, errs = rootApplier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
}

func TestDisabledResource(t *testing.T) {
	if !*e2e.E2E || !*e2e.MultiRepo {
		return
	}
	prepare(t)
	t.Log("namespace applier: first resource apply, then disable the resource.")
	resources := []core.Object{
		fake.ConfigMapObject(core.Name(testCM1), core.Namespace(metav1.NamespaceDefault), syncertest.ManagementEnabled),
	}
	wantGVKs := map[schema.GroupVersionKind]struct{}{
		kinds.ConfigMap(): {},
	}
	applier := kptapplier.NewNamespaceApplier(k8sClient, declared.Scope(metav1.NamespaceDefault))
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
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// disable the object
	resources = []core.Object{
		fake.ConfigMapObject(core.Name(testCM1), core.Namespace(metav1.NamespaceDefault), syncertest.ManagementDisabled),
	}
	_, errs = applier.Apply(ctx, resources)
	if err != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm testCM1 exists and doesn't contain nomos metadata.
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: metav1.NamespaceDefault,
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
		Namespace: metav1.NamespaceDefault,
	}, cmObject)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	t.Log("namespace applier: applier can manage the disabled resource again when enabling")
	resources = []core.Object{
		fake.ConfigMapObject(core.Name(testCM1), core.Namespace(metav1.NamespaceDefault), syncertest.ManagementEnabled),
	}
	_, errs = applier.Apply(ctx, resources)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
	// confirm testCM1 exists
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      testCM1,
		Namespace: metav1.NamespaceDefault,
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

	// cleanup the applied resources
	_, errs = applier.Apply(ctx, nil)
	if errs != nil {
		t.Fatalf("unexpected error %v", errs)
	}
}

func getCRDV1beta1() (*unstructured.Unstructured, error) {
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

func getCRDV1() (*unstructured.Unstructured, error) {
	crdManifest := `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: examples.test.io
spec:
  group: test.io
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: string
  scope: Namespaced
  names:
    plural: examples
    singular: example
    kind: Example
    shortNames:
    - ex
`

	crd := &unstructured.Unstructured{}
	err := yaml.Unmarshal([]byte(crdManifest), crd)
	return crd, err
}
