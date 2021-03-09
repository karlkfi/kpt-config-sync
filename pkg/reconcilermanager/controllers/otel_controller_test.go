package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metrics"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"golang.org/x/oauth2/google"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	depAnnotation       = "6542d3f02b02979cd5322ae60b20d0d9"
	depAnnotationCustom = "a5b98e40aa7ae6bd2326dca598900f83"
)

func setupOtelReconciler(t *testing.T, objs ...client.Object) (*syncerFake.Client, *OtelReconciler) {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := appsv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}

	fakeClient := syncerFake.NewClient(t, s, objs...)
	testReconciler := NewOtelReconciler(
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("Otel"),
		s,
	)
	return fakeClient, testReconciler
}

func TestOtelReconciler(t *testing.T) {
	cm := configMapWithData(
		metrics.MonitoringNamespace,
		metrics.OtelCollectorName,
		map[string]string{"otel-collector-config.yaml": ""},
	)
	reqNamespacedName := namespacedName(metrics.OtelCollectorName, metrics.MonitoringNamespace)
	fakeClient, testReconciler := setupOtelReconciler(t, cm, fake.DeploymentObject(core.Name(metrics.OtelCollectorName), core.Namespace(metrics.MonitoringNamespace)))

	getDefaultCredentials = func(ctx context.Context) (*google.Credentials, error) {
		return nil, errors.New("could not find default credentials")
	}

	// Test updating Configmap and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantDeployment := fake.DeploymentObject(
		core.Namespace(metrics.MonitoringNamespace),
		core.Name(metrics.OtelCollectorName),
	)

	// compare ConfigMap. Expect no change in the ConfigMap.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ConfigMap diff %s", diff)
	}

	// compare Deployment annotation. Expect no annotation.
	gotDeployment := fakeClient.Objects[core.IDOf(wantDeployment)].(*appsv1.Deployment)
	if diff := cmp.Diff(gotDeployment.Spec.Template.Annotations, wantDeployment.Spec.Template.Annotations, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Deployment diff %s", diff)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestOtelReconcilerStackdriver(t *testing.T) {
	cm := configMapWithData(
		metrics.MonitoringNamespace,
		metrics.OtelCollectorName,
		map[string]string{"otel-collector-config.yaml": ""},
	)
	reqNamespacedName := namespacedName(metrics.OtelCollectorName, metrics.MonitoringNamespace)
	fakeClient, testReconciler := setupOtelReconciler(t, cm, fake.DeploymentObject(core.Name(metrics.OtelCollectorName), core.Namespace(metrics.MonitoringNamespace)))

	getDefaultCredentials = func(ctx context.Context) (*google.Credentials, error) {
		return &google.Credentials{
			ProjectID:   "test",
			TokenSource: nil,
			JSON:        nil,
		}, nil
	}

	// Test updating Configmap and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := configMapWithData(
		metrics.MonitoringNamespace,
		metrics.OtelCollectorName,
		map[string]string{"otel-collector-config.yaml": metrics.CollectorConfigStackdriver},
	)

	wantDeployment := fake.DeploymentObject(
		core.Namespace(metrics.MonitoringNamespace),
		core.Name(metrics.OtelCollectorName),
	)
	core.SetAnnotation(&wantDeployment.Spec.Template, v1alpha1.ConfigMapAnnotationKey, depAnnotation)

	// compare ConfigMap
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantConfigMap)], wantConfigMap, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ConfigMap diff %s", diff)
	}

	// compare Deployment annotation
	gotDeployment := fakeClient.Objects[core.IDOf(wantDeployment)].(*appsv1.Deployment)
	if diff := cmp.Diff(gotDeployment.Spec.Template.Annotations, wantDeployment.Spec.Template.Annotations, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Deployment diff %s", diff)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestOtelReconcilerCustom(t *testing.T) {
	cm := configMapWithData(
		metrics.MonitoringNamespace,
		metrics.OtelCollectorName,
		map[string]string{"otel-collector-config.yaml": ""},
	)
	cmCustom := configMapWithData(
		metrics.MonitoringNamespace,
		metrics.OtelCollectorCustomCM,
		map[string]string{"otel-collector-config.yaml": "custom"},
	)
	reqNamespacedName := namespacedName(metrics.OtelCollectorCustomCM, metrics.MonitoringNamespace)
	fakeClient, testReconciler := setupOtelReconciler(t, cm, cmCustom, fake.DeploymentObject(core.Name(metrics.OtelCollectorName), core.Namespace(metrics.MonitoringNamespace)))

	getDefaultCredentials = func(ctx context.Context) (*google.Credentials, error) {
		return nil, nil
	}

	// Test updating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantDeployment := fake.DeploymentObject(
		core.Namespace(metrics.MonitoringNamespace),
		core.Name(metrics.OtelCollectorName),
	)
	core.SetAnnotation(&wantDeployment.Spec.Template, v1alpha1.ConfigMapAnnotationKey, depAnnotationCustom)

	// compare Deployment annotation
	gotDeployment := fakeClient.Objects[core.IDOf(wantDeployment)].(*appsv1.Deployment)
	if diff := cmp.Diff(gotDeployment.Spec.Template.Annotations, wantDeployment.Spec.Template.Annotations, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Deployment diff %s", diff)
	}
	t.Log("Deployment successfully updated")
}
