package reconcile

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	e := v1.ErrorResource{
		SourcePath:        ns.GetAnnotations()[v1.SourcePathAnnotationKey],
		ResourceName:      ns.GetName(),
		ResourceNamespace: ns.GetNamespace(),
		ResourceGVK:       ns.GroupVersionKind(),
	}
	cme := v1.ConfigManagementError{
		ErrorMessage: errMsg,
	}
	cme.ErrorResources = append(cme.ErrorResources, e)
	return cme
}

// SetClusterConfigStatus updates the status sub-resource of the ClusterConfig based on reconciling the ClusterConfig.
func SetClusterConfigStatus(ctx context.Context, client *client.Client, config *v1.ClusterConfig, now func() metav1.Time,
	errs ...v1.ConfigManagementError) status.Error {
	freshSyncToken := config.Status.Token == config.Spec.Token
	if config.Status.SyncState.IsSynced() && freshSyncToken && len(errs) == 0 {
		glog.Infof("Status for ClusterConfig %q is already up-to-date.", config.Name)
		return nil
	}

	updateFn := func(obj runtime.Object) (runtime.Object, error) {
		newConfig := obj.(*v1.ClusterConfig)
		newConfig.Status.Token = config.Spec.Token
		newConfig.Status.SyncTime = now()
		newConfig.Status.SyncErrors = errs
		if len(errs) > 0 {
			newConfig.Status.SyncState = v1.StateError
		} else {
			newConfig.Status.SyncState = v1.StateSynced
		}
		return newConfig, nil
	}
	_, err := client.UpdateStatus(ctx, config, updateFn)
	return err
}
