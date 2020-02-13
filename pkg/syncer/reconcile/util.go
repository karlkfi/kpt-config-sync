package reconcile

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

	updateFn := func(obj core.Object) (core.Object, error) {
		newConfig := obj.(*v1.ClusterConfig)
		newConfig.Status.Token = config.Spec.Token
		newConfig.Status.SyncTime = now()
		newConfig.Status.SyncErrors = errs
		if len(errs) > 0 {
			newConfig.Status.SyncState = v1.StateError
		} else {
			newConfig.Status.SyncState = v1.StateSynced
		}

		newConfig.Status.ResourceConditions = config.Status.ResourceConditions
		return newConfig, nil
	}
	_, err := client.UpdateStatus(ctx, config, updateFn)
	return err
}

// AnnotationsHaveResourceCondition checks if the given annotations contain at least one resource condition
func AnnotationsHaveResourceCondition(annotations map[string]string) bool {
	if _, ok := annotations[v1.ResourceStatusErrorsKey]; ok {
		return true
	}
	if _, ok := annotations[v1.ResourceStatusUnreadyKey]; ok {
		return true
	}
	return false
}

// MakeResourceCondition makes a resource condition from an unstructured object and the given config token
func MakeResourceCondition(obj unstructured.Unstructured, token string) v1.ResourceCondition {
	resourceCondition := v1.ResourceCondition{ResourceState: v1.ResourceStateHealthy, Token: token}
	resourceCondition.GroupVersionKind = obj.GroupVersionKind().String()
	resourceCondition.NamespacedName = fmt.Sprintf("%v/%v", obj.GetNamespace(), obj.GetName())

	if unready, ok := obj.GetAnnotations()[v1.ResourceStatusUnreadyKey]; ok {
		resourceCondition.ResourceState = v1.ResourceStateUnready
		resourceCondition.UnreadyReasons = append(resourceCondition.UnreadyReasons, unready)
	}
	if errors, ok := obj.GetAnnotations()[v1.ResourceStatusErrorsKey]; ok {
		resourceCondition.ResourceState = v1.ResourceStateError
		resourceCondition.Errors = append(resourceCondition.Errors, errors)
	}
	return resourceCondition
}

func filterContextCancelled(err error) error {
	if errs, ok := err.(status.MultiError); ok {
		if len(errs.Errors()) == 1 {
			err = errs.Errors()[0]
		} else {
			return filterMultiErrorContextCancelled(errs)
		}
	}
	c := errors.Cause(err)
	if reflect.DeepEqual(c, context.Canceled) {
		return nil
	}
	// http client errors don't implement causer. The underlying error is in one of the struct's fields.
	if ue, ok := c.(*url.Error); ok && reflect.DeepEqual(ue.Err, context.Canceled) {
		return nil
	}
	return err
}

func filterMultiErrorContextCancelled(errs status.MultiError) status.MultiError {
	var filtered status.MultiError
	for _, e := range errs.Errors() {
		if fe := filterContextCancelled(e); fe != nil {
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
