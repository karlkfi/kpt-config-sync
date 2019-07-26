package reconcile

import (
	"context"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/cache"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/syncer/differ"
	"github.com/google/nomos/pkg/syncer/metrics"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ reconcile.Reconciler = &ClusterConfigReconciler{}

// ClusterConfigReconciler reconciles a ClusterConfig object.
type ClusterConfigReconciler struct {
	client   *client.Client
	applier  Applier
	cache    cache.GenericCache
	recorder record.EventRecorder
	decoder  decode.Decoder
	toSync   []schema.GroupVersionKind
	now      func() metav1.Time
	// A cancelable ambient context for all reconciler operations.
	ctx context.Context
}

// NewClusterConfigReconciler returns a new ClusterConfigReconciler.  ctx is the ambient context
// to use for all reconciler operations.
func NewClusterConfigReconciler(ctx context.Context, client *client.Client, applier Applier, cache cache.GenericCache, recorder record.EventRecorder,
	decoder decode.Decoder, now func() metav1.Time, toSync []schema.GroupVersionKind) *ClusterConfigReconciler {
	return &ClusterConfigReconciler{
		client:   client,
		applier:  applier,
		cache:    cache,
		recorder: recorder,
		decoder:  decoder,
		toSync:   toSync,
		now:      now,
		ctx:      ctx,
	}
}

// Reconcile is the Reconcile callback for ClusterConfigReconciler.
func (r *ClusterConfigReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if request.Name == v1.CRDClusterConfigName {
		// We handle the CRD Cluster Config in the CRD controller, so don't reconcile it here.
		return reconcile.Result{}, nil
	}

	start := r.now()
	metrics.ReconcileEventTimes.WithLabelValues("cluster").Set(float64(start.Unix()))

	ctx, cancel := context.WithTimeout(r.ctx, reconcileTimeout)
	defer cancel()

	err := r.reconcileConfig(ctx, request.NamespacedName)
	metrics.ReconcileDuration.WithLabelValues("cluster", metrics.StatusLabel(err)).Observe(time.Since(start.Time).Seconds())

	return reconcile.Result{}, err
}

func (r *ClusterConfigReconciler) reconcileConfig(ctx context.Context, name types.NamespacedName) error {
	clusterConfig := &v1.ClusterConfig{}
	err := r.cache.Get(ctx, name, clusterConfig)
	if err != nil {
		err = errors.Wrapf(err, "could not retrieve clusterconfig %q", name)
		glog.Error(err)
		return err
	}
	clusterConfig.SetGroupVersionKind(kinds.ClusterConfig())

	if name.Name != v1.ClusterConfigName {
		err := errors.Errorf("ClusterConfig resource has invalid name %q. To fix, delete the ClusterConfig.", name.Name)
		r.recorder.Eventf(clusterConfig, corev1.EventTypeWarning, "InvalidClusterConfig", err.Error())
		glog.Warning(err)
		// Update the status on a best effort basis. We don't want to retry handling a ClusterConfig
		// we want to ignore and it's possible it has been deleted by the time we reconcile it.
		if err2 := SetClusterConfigStatus(ctx, r.client, clusterConfig, r.now, NewConfigManagementError(clusterConfig, err)); err2 != nil {
			r.recorder.Eventf(clusterConfig, corev1.EventTypeWarning, "StatusUpdateFailed",
				"failed to update cluster config status: %v", err2)
		}
		return nil
	}

	rErr := r.manageConfigs(ctx, clusterConfig)
	// Filter out errors caused by a context cancellation. These errors are expected and uninformative.
	if filtered := filterWithCause(rErr, context.Canceled); filtered != nil {
		glog.Errorf("Could not reconcile clusterconfig: %v", filtered)
	}
	return rErr
}

func (r *ClusterConfigReconciler) manageConfigs(ctx context.Context, config *v1.ClusterConfig) error {
	if gks := resourcesWithoutSync(config.Spec.Resources, r.toSync); gks != nil {
		glog.Infof(
			"ClusterConfigReconciler encountered "+
				"group-kind(s) %s that were not present in a sync, waiting for reconciler restart",
			strings.Join(gks, ", "))
		// We only reach this case on a race condition where the reconciler is run before the
		// changes to Sync objects are picked up.  We exit early since there are resources we can't
		// properly handle which will cause status on the ClusterConfig to incorrectly report that
		// everything is fully synced.  We log info and return nil here since the Sync metacontroller
		// will restart this reconciler shortly.
		return nil
	}

	grs, err := r.decoder.DecodeResources(config.Spec.Resources...)
	if err != nil {
		return errors.Wrapf(err, "could not process cluster config: %q", config.GetName())
	}

	var errBuilder status.MultiError
	reconcileCount := 0
	for _, gvk := range r.toSync {
		declaredInstances := grs[gvk]
		for _, decl := range declaredInstances {
			SyncedAt(decl, config.Spec.Token)
		}

		actualInstances, err := r.cache.UnstructuredList(gvk, "")
		if err != nil {
			errBuilder = status.Append(errBuilder, status.APIServerWrapf(err, "failed to list from config controller for %q", gvk))
			continue
		}

		allDeclaredVersions := AllVersionNames(grs, gvk.GroupKind())
		diffs := differ.Diffs(declaredInstances, actualInstances, allDeclaredVersions)
		for _, diff := range diffs {
			if updated, err := HandleDiff(ctx, r.applier, diff, r.recorder); err != nil {
				errBuilder = status.Append(errBuilder, err)
			} else if updated {
				reconcileCount++
			}
		}
	}
	if err := SetClusterConfigStatus(ctx, r.client, config, r.now, status.ToCME(errBuilder)...); err != nil {
		errBuilder = status.Append(errBuilder, err)
		r.recorder.Eventf(config, corev1.EventTypeWarning, "StatusUpdateFailed",
			"failed to update cluster config status: %v", err)
	}
	if errBuilder == nil && reconcileCount > 0 {
		r.recorder.Eventf(config, corev1.EventTypeNormal, "ReconcileComplete",
			"cluster config was successfully reconciled: %d changes", reconcileCount)
	}
	return errBuilder
}

// NewConfigManagementError returns a ConfigManagementError corresponding to the given ClusterConfig and error.
func NewConfigManagementError(config *v1.ClusterConfig, err error) v1.ConfigManagementError {
	e := v1.ErrorResource{
		SourcePath:        config.GetAnnotations()[v1.SourcePathAnnotationKey],
		ResourceName:      config.GetName(),
		ResourceNamespace: config.GetNamespace(),
		ResourceGVK:       config.GroupVersionKind(),
	}
	cme := v1.ConfigManagementError{
		ErrorMessage: err.Error(),
	}
	cme.ErrorResources = append(cme.ErrorResources, e)
	return cme
}

// removeEmptyRulesField removes the Rules field from ClusterRole when it's an empty list.
// This is to ensure that we don't overwrite PolicyRules generated by other controllers
// for aggregated ClusterRoles when we `apply` changes.
func removeEmptyRulesField(u *unstructured.Unstructured) {
	if u == nil {
		// Nothing to do.
		return
	}
	if u.GroupVersionKind() != kinds.ClusterRole() {
		return
	}

	if rules, ok := u.Object["rules"]; ok && (rules == nil || len(rules.([]interface{})) == 0) {
		delete(u.Object, "rules")
	}
}
