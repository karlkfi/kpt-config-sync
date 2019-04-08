package reconcile

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AllVersionNames returns the set of names of all resources with the specified GroupKind.
func AllVersionNames(resources map[schema.GroupVersionKind][]*unstructured.Unstructured, gk schema.GroupKind) map[string]bool {
	names := map[string]bool{}
	for gvk, rs := range resources {
		if gvk.GroupKind() != gk {
			continue
		}
		for _, r := range rs {
			n := r.GetName()
			if names[n] {
				panic(fmt.Errorf("duplicate resources names %q declared for %s", n, gvk))
			} else {
				names[n] = true
			}
		}
	}
	return names
}

// cmeForNamespace returns a ConfigManagementError for the given Namespace and error message.
func cmeForNamespace(ns *corev1.Namespace, errMsg string) v1.ConfigManagementError {
	return v1.ConfigManagementError{
		SourcePath:        ns.GetAnnotations()[v1.SourcePathAnnotationKey],
		ResourceName:      ns.GetName(),
		ResourceNamespace: ns.GetNamespace(),
		ResourceGVK:       ns.GroupVersionKind(),
		ErrorMessage:      errMsg,
	}
}

// CmesForResourceError returns ConfigManagementErrors built from the given ResourceError.
func CmesForResourceError(resErr status.ResourceError) []v1.ConfigManagementError {
	resCount := len(resErr.Resources())
	if resCount == 0 {
		return []v1.ConfigManagementError{
			{ErrorMessage: resErr.Error()},
		}
	}

	configErrs := make([]v1.ConfigManagementError, resCount)
	for i, res := range resErr.Resources() {
		configErrs[i] = v1.ConfigManagementError{
			SourcePath:        res.SlashPath(),
			ResourceName:      res.Name(),
			ResourceNamespace: res.Namespace(),
			ResourceGVK:       res.GroupVersionKind(),
			ErrorMessage:      resErr.Error(),
		}
	}
	return configErrs
}

// SetClusterConfigStatus updates the status sub-resource of the ClusterConfig based on reconciling the ClusterConfig.
func SetClusterConfigStatus(ctx context.Context, client *client.Client, policy *v1.ClusterConfig,
	errs ...v1.ConfigManagementError) status.ResourceError {
	freshSyncToken := policy.Status.Token == policy.Spec.Token
	if policy.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for ClusterConfig %q is already up-to-date.", policy.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newPolicy := obj.(*v1.ClusterConfig)
		newPolicy.Status.Token = policy.Spec.Token
		newPolicy.Status.SyncTime = now()
		newPolicy.Status.SyncErrors = errs
		if len(errs) > 0 {
			newPolicy.Status.SyncState = v1.StateError
		} else {
			newPolicy.Status.SyncState = v1.StateSynced
		}
		return newPolicy, nil
	}
	_, err := client.UpdateStatus(ctx, policy, updateFn)
	if err != nil {
		metrics.ErrTotal.WithLabelValues("", policy.GroupVersionKind().Kind, "update").Inc()
	}
	return err
}
