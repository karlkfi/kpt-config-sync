package nomostest

import (
	"fmt"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Predicate evaluates a client.Object, returning an error if it fails validation.
type Predicate func(o client.Object) error

// ErrWrongType indicates that the caller passed an object of the incorrect type
// to the Predicate.
var ErrWrongType = errors.New("wrong type")

// WrongTypeErr reports that the passed type was not equivalent to the wanted
// type.
func WrongTypeErr(got, want interface{}) error {
	return fmt.Errorf("%w: got %T, want %T", ErrWrongType, got, want)
}

// ErrFailedPredicate indicates the the object on the API server does not match
// the Predicate.
var ErrFailedPredicate = errors.New("failed predicate")

// HasAnnotation returns a predicate that tests if an Object has the specified
// annotation key/value pair.
func HasAnnotation(key, value string) Predicate {
	return func(o client.Object) error {
		got, ok := o.GetAnnotations()[key]
		if !ok {
			return fmt.Errorf("object %q does not have annotation %q; want %q", o.GetName(), key, value)
		}
		if got != value {
			return fmt.Errorf("got %q for annotation %q on object %q; want %q", got, key, o.GetName(), value)
		}
		return nil
	}
}

// HasAnnotationKey returns a predicate that tests if an Object has the specified
// annotation key.
func HasAnnotationKey(key string) Predicate {
	return func(o client.Object) error {
		_, ok := o.GetAnnotations()[key]
		if !ok {
			return fmt.Errorf("object %q does not have annotation %q", o.GetName(), key)
		}
		return nil
	}
}

// HasAllAnnotationKeys returns a predicate that tests if an Object has the specified
// annotation keys.
func HasAllAnnotationKeys(keys ...string) Predicate {
	return func(o client.Object) error {
		for _, key := range keys {
			predicate := HasAnnotationKey(key)

			err := predicate(o)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

// MissingAnnotation returns a predicate that tests that an object does not have
// a specified annotation.
func MissingAnnotation(key string) Predicate {
	return func(o client.Object) error {
		_, ok := o.GetAnnotations()[key]
		if ok {
			return fmt.Errorf("object %v has annotation %s, want missing", o.GetName(), key)
		}
		return nil
	}
}

// HasLabel returns a predicate that tests if an Object has the specified label key/value pair.
func HasLabel(key, value string) Predicate {
	return func(o client.Object) error {
		got, ok := o.GetLabels()[key]
		if !ok {
			return fmt.Errorf("object %q does not have label %q; wanted %q", o.GetName(), key, value)
		}
		if got != value {
			return fmt.Errorf("got %q for label %q on object %q; wanted %q", got, key, o.GetName(), value)
		}
		return nil
	}
}

// MissingLabel returns a predicate that tests that an object does not have
// a specified label.
func MissingLabel(key string) Predicate {
	return func(o client.Object) error {
		_, ok := o.GetLabels()[key]
		if ok {
			return fmt.Errorf("object %v has label %s, want missing", o.GetName(), key)
		}
		return nil
	}
}

// HasExactlyAnnotationKeys ensures the Object has exactly the passed set of
// annotations, ignoring values.
func HasExactlyAnnotationKeys(wantKeys ...string) Predicate {
	sort.Strings(wantKeys)
	return func(o client.Object) error {
		annotations := o.GetAnnotations()
		var gotKeys []string
		for k := range annotations {
			gotKeys = append(gotKeys, k)
		}
		sort.Strings(gotKeys)
		if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
			return errors.Errorf("unexpected diff in metadata.annotation keys: %s", diff)
		}
		return nil
	}
}

// HasExactlyLabelKeys ensures the Object has exactly the passed set of
// labels, ignoring values.
func HasExactlyLabelKeys(wantKeys ...string) Predicate {
	wantKeys = append(wantKeys, TestLabel)
	sort.Strings(wantKeys)
	return func(o client.Object) error {
		labels := o.GetLabels()
		var gotKeys []string
		for k := range labels {
			gotKeys = append(gotKeys, k)
		}
		sort.Strings(gotKeys)
		if diff := cmp.Diff(wantKeys, gotKeys); diff != "" {
			return errors.Errorf("unexpected diff in metadata.annotation keys: %s", diff)
		}
		return nil
	}
}

// HasKeyValuePairInConfigMapData verifies that a ConfigMap object has a key/value pair.
func HasKeyValuePairInConfigMapData(k, v string) Predicate {
	return func(o client.Object) error {
		cm, ok := o.(*corev1.ConfigMap)
		if !ok {
			return WrongTypeErr(cm, &corev1.ConfigMap{})
		}
		valueInCM, ok := cm.Data[k]
		if !ok {
			return errors.Errorf("The Data field of the %q ConfigMap does not have key %q.", core.GKNN(cm), k)
		}
		if valueInCM != v {
			return errors.Errorf("The value for the %q key in the Data field of the %q ConfigMap should be %q, got %q.", k, core.GKNN(cm), v, valueInCM)
		}
		return nil
	}
}

// MissingKeyInConfigMapData verifies that a ConfigMap object does not have a key.
func MissingKeyInConfigMapData(k string) Predicate {
	return func(o client.Object) error {
		cm, ok := o.(*corev1.ConfigMap)
		if !ok {
			return WrongTypeErr(cm, &corev1.ConfigMap{})
		}
		if _, ok := cm.Data[k]; ok {
			return errors.Errorf("The Data field of the %q ConfigMap should not have key %q.", core.GKNN(cm), k)
		}
		return nil
	}
}

// HasCorrectResourceLimits verify a root/namespace reconciler has the correct resource limits.
func HasCorrectResourceLimits(reconcilerCPULimits, reconcilerMemLimits, gitSyncCPULimits, gitSyncMemLimits resource.Quantity) Predicate {
	return func(o client.Object) error {
		dep, ok := o.(*appsv1.Deployment)
		if !ok {
			return WrongTypeErr(dep, &appsv1.Deployment{})
		}
		for _, container := range dep.Spec.Template.Spec.Containers {
			switch container.Name {
			case reconcilermanager.Reconciler:
				if container.Resources.Limits[corev1.ResourceCPU] != reconcilerCPULimits {
					return errors.Errorf("The CPU limit of the %q container should be %v, got %v", container.Name, reconcilerCPULimits, container.Resources.Limits[corev1.ResourceCPU])
				}
				if container.Resources.Limits[corev1.ResourceMemory] != reconcilerMemLimits {
					return errors.Errorf("The memory limit of the %q container should be %v, got %v", container.Name, reconcilerMemLimits, container.Resources.Limits[corev1.ResourceMemory])
				}

			case reconcilermanager.GitSync:
				if container.Resources.Limits[corev1.ResourceCPU] != gitSyncCPULimits {
					return errors.Errorf("The CPU limit of the %q container should be %v, got %v", container.Name, gitSyncCPULimits, container.Resources.Limits[corev1.ResourceCPU])
				}
				if container.Resources.Limits[corev1.ResourceMemory] != gitSyncMemLimits {
					return errors.Errorf("The memory limit of the %q container should be %v, got %v", container.Name, gitSyncMemLimits, container.Resources.Limits[corev1.ResourceMemory])
				}
			}
		}
		return nil
	}
}

// NotPendingDeletion ensures o is not pending deletion.
//
// Check this when the object could be scheduled for deletion, to avoid flaky
// behavior when we're ensuring we don't want something to be deleted.
func NotPendingDeletion(o client.Object) error {
	if o.GetDeletionTimestamp() == nil {
		return nil
	}
	return errors.Errorf("object has non-nil deletionTimestamp")
}

// HasAllNomosMetadata ensures that the object contains the expected
// nomos labels and annotations.
func HasAllNomosMetadata(multiRepo bool) Predicate {
	return func(o client.Object) error {
		annotationKeys := metadata.GetNomosAnnotationKeys(multiRepo)
		labels := metadata.SyncerLabels()

		predicates := []Predicate{HasAllAnnotationKeys(annotationKeys...), HasAnnotation("configmanagement.gke.io/managed", "enabled")}
		for labelKey, value := range labels {
			predicates = append(predicates, HasLabel(labelKey, value))
		}

		for _, predicate := range predicates {
			err := predicate(o)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

// NoConfigSyncMetadata ensures that the object doesn't
// contain configsync labels and annotations.
func NoConfigSyncMetadata() Predicate {
	return func(o client.Object) error {
		if metadata.HasConfigSyncMetadata(o) {
			return fmt.Errorf("object %q shouldn't have configsync metadta %v, %v", o.GetName(), o.GetLabels(), o.GetAnnotations())
		}
		return nil
	}
}

// AllResourcesAreCurrent ensures that the managed resources
// are all Current in the ResourceGroup CR.
func AllResourcesAreCurrent() Predicate {
	return func(o client.Object) error {
		u, ok := o.(*unstructured.Unstructured)
		if !ok {
			return WrongTypeErr(u, &unstructured.Unstructured{})
		}
		resourceStatuses, found, err := unstructured.NestedSlice(u.Object, "status", "resourceStatuses")
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("resource status not found in %v", u)
		}
		for _, resource := range resourceStatuses {
			s, ok := resource.(map[string]interface{})
			if !ok {
				return WrongTypeErr(s, map[string]interface{}{})
			}
			status, found, err := unstructured.NestedString(s, "status")
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("status field not found for resource %v", resource)
			}
			if status != "Current" {
				return fmt.Errorf("status %v is not Current", status)
			}
		}
		return nil
	}
}
