package controllers

import (
	"context"
	"fmt"
	"reflect"

	traceapi "cloud.google.com/go/trace/apiv2"
	"github.com/go-logr/logr"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/status"
	"golang.org/x/oauth2/google"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &OtelReconciler{}

// OtelReconciler reconciles OpenTelemetry ConfigMaps.
type OtelReconciler struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

// NewOtelReconciler returns a new OtelReconciler.
func NewOtelReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *OtelReconciler {
	return &OtelReconciler{
		client: client,
		log:    log,
		scheme: scheme,
	}
}

// Reconcile the otel ConfigMap and update the Deployment annotation.
func (r *OtelReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.WithValues("otel", req.NamespacedName)

	configMapDataHash, err := r.reconcileConfigMap(ctx, req)
	if err != nil {
		log.Error(err, "Failed to create/update ConfigMap")
		return controllerruntime.Result{}, err
	}
	err = r.updateDeploymentAnnotation(ctx, configMapDataHash)
	if err != nil {
		log.Error(err, "Failed to update Deployment")
		return controllerruntime.Result{}, err
	}
	return controllerruntime.Result{}, nil
}

// reconcileConfigMap reconciles ConfigMaps declared in the `config-management-monitoring`
// namespace and returns its hash.
//
// If the reconciled ConfigMap is the standard `otel-collector` map, we check
// whether Application Default Credentials exist. If so, we patch the map with a
// collector config that includes both a Prometheus and a Stackdriver exporter.
func (r *OtelReconciler) reconcileConfigMap(ctx context.Context, req reconcile.Request) ([]byte, error) {
	// The otel-collector Deployment only reads from the `otel-collector` and
	// `otel-collector-custom` ConfigMaps, so we only reconcile these two maps.
	if req.Name != metrics.OtelCollectorName && req.Name != metrics.OtelCollectorCustomCM {
		return nil, nil
	}

	var cm corev1.ConfigMap
	if err := r.client.Get(ctx, req.NamespacedName, &cm); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, status.APIServerErrorf(err, "failed to get otel ConfigMap %s", req.NamespacedName.String())
	}
	if cm.Name == metrics.OtelCollectorName {
		return r.patchStandardConfigMap(ctx, &cm)
	}
	return hash(cm)
}

// patchStandardConfigMap patches the standard `otel-collector` ConfigMap to
// ensure the Prometheus exporter is enabled. It automatically enables Stackdriver
// as well if Application Default Credentials are present.
func (r *OtelReconciler) patchStandardConfigMap(ctx context.Context, cm *corev1.ConfigMap) ([]byte, error) {
	existing := cm.DeepCopy()
	patch := client.MergeFrom(existing)
	if _, ok := cm.Data["otel-collector-config.yaml"]; ok {
		creds, _ := getDefaultCredentials(ctx)
		if creds != nil && creds.ProjectID != "" {
			cm.Data["otel-collector-config.yaml"] = metrics.CollectorConfigStackdriver
		}
	}
	if reflect.DeepEqual(existing, cm) {
		return nil, nil
	}

	if err := r.client.Patch(ctx, cm, patch); err != nil {
		return nil, err
	}
	r.log.Info("ConfigMap successfully reconciled", operationSubjectName, metrics.OtelCollectorName)
	return hash(cm)
}

// updateDeploymentAnnotation updates the otel deployment's spec.template.annotation
// with the otel ConfigMap hash. This triggers the deployment to restart in the
// event of a ConfigMap update.
func (r *OtelReconciler) updateDeploymentAnnotation(ctx context.Context, hash []byte) error {
	if hash == nil {
		return nil
	}

	var dep appsv1.Deployment
	dep.Name = metrics.OtelCollectorName
	dep.Namespace = metrics.MonitoringNamespace
	key := client.ObjectKeyFromObject(&dep)

	if err := r.client.Get(ctx, key, &dep); err != nil {
		return status.APIServerError(err, "failed to get otel Deployment")
	}

	existing := dep.DeepCopy()
	patch := client.MergeFrom(existing)

	// Mutate Annotation with the hash of configmap.data from the otel ConfigMap
	// creates/updates.
	core.SetAnnotation(&dep.Spec.Template, v1alpha1.ConfigMapAnnotationKey, fmt.Sprintf("%x", hash))

	if reflect.DeepEqual(existing, dep) {
		return nil
	}

	if err := r.client.Patch(ctx, &dep, patch); err != nil {
		return err
	}
	return nil
}

// SetupWithManager registers otel controller with reconciler-manager.
func (r *OtelReconciler) SetupWithManager(mgr controllerruntime.Manager) error {
	// Process create / update events for resources in the `config-management-monitoring` namespace.
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetNamespace() == metrics.MonitoringNamespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetNamespace() == metrics.MonitoringNamespace
		},
	}
	return controllerruntime.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(p)).
		Complete(r)
}

// getDefaultCredentials searches for "Application Default Credentials":
// https://developers.google.com/accounts/docs/application-default-credentials.
// It can be overridden during tests.
var getDefaultCredentials = func(ctx context.Context) (*google.Credentials, error) {
	return google.FindDefaultCredentials(ctx, traceapi.DefaultAuthScopes()...)
}
