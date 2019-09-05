package reconcile

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	"github.com/pkg/errors"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/client"
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

func filterWithCause(err error, cause error) error {
	if errs, ok := err.(status.MultiError); ok {
		if len(errs.Errors()) == 1 {
			err = errs.Errors()[0]
		} else {
			return filterMultiErrorWithCause(errs, cause)
		}
	}
	c := errors.Cause(err)
	if reflect.DeepEqual(c, cause) {
		return nil
	}
	// http client errors don't implement causer. The underlying error is in one of the struct's fields.
	if ue, ok := c.(*url.Error); ok && reflect.DeepEqual(ue.Err, cause) {
		return nil
	}
	return err
}

func filterMultiErrorWithCause(errs status.MultiError, cause error) status.MultiError {
	var filtered status.MultiError
	for _, e := range errs.Errors() {
		if fe := filterWithCause(e, cause); fe != nil {
			filtered = status.Append(filtered, fe)
		}
	}
	return filtered
}

// resourcesWithoutSync returns a list of strings representing all group/kind in the
// v1.GenericResources list that are not found in a Sync.
func resourcesWithoutSync(
	resources []v1.GenericResources, toSync []schema.GroupVersionKind) []string {
	declaredGroupKind := map[schema.GroupKind]struct{}{}
	for _, res := range resources {
		declaredGroupKind[schema.GroupKind{Group: res.Group, Kind: res.Kind}] = struct{}{}
	}
	for _, gvk := range toSync {
		delete(declaredGroupKind, gvk.GroupKind())
	}

	var gks []string
	for gk := range declaredGroupKind {
		gks = append(gks, gk.String())
	}
	return gks
}
