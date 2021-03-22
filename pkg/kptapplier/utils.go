package kptapplier

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	syncerreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// OwningInventoryKey is the annotation key for marking the owning-inventory object.
	// TODO(jingfangliu): Convert this key to a constant from the apply library.
	OwningInventoryKey = "config.k8s.io/owning-inventory"
)

func partitionObjs(objs []client.Object) ([]client.Object, []client.Object) {
	var enabled []client.Object
	var disabled []client.Object
	for _, obj := range objs {
		if obj.GetAnnotations()[v1.ResourceManagementKey] == v1.ResourceManagementDisabled {
			disabled = append(disabled, obj)
		} else {
			enabled = append(enabled, obj)
		}
	}
	return enabled, disabled
}

func toUnstructured(objs []client.Object) ([]*unstructured.Unstructured, status.MultiError) {
	var errs status.MultiError
	var unstructureds []*unstructured.Unstructured
	for _, obj := range objs {
		u, err := syncerreconcile.AsUnstructuredSanitized(obj)
		if err != nil {
			// This should never happen.
			errs = status.Append(errs, status.InternalErrorBuilder.Wrap(err).
				Sprintf("converting %v to unstructured.Unstructured", core.IDOf(obj)).Build())
		}
		unstructureds = append(unstructureds, u)
	}
	return unstructureds, errs
}

func objMetaFrom(obj client.Object) object.ObjMetadata {
	return object.ObjMetadata{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		GroupKind: obj.GetObjectKind().GroupVersionKind().GroupKind(),
	}
}

func idFrom(identifier object.ObjMetadata) core.ID {
	return core.ID{
		GroupKind: identifier.GroupKind,
		ObjectKey: client.ObjectKey{
			Name:      identifier.Name,
			Namespace: identifier.Namespace,
		},
	}
}

func removeFrom(all []object.ObjMetadata, toRemove []client.Object) []object.ObjMetadata {
	m := map[object.ObjMetadata]bool{}
	for _, a := range all {
		m[a] = true
	}

	for _, r := range toRemove {
		meta := object.ObjMetadata{
			Namespace: r.GetNamespace(),
			Name:      r.GetName(),
			GroupKind: r.GetObjectKind().GroupVersionKind().GroupKind(),
		}
		delete(m, meta)
	}
	var results []object.ObjMetadata
	for key := range m {
		results = append(results, key)
	}
	return results
}

func removeConfigSyncLabelsAndAnnotations(obj *unstructured.Unstructured) (map[string]string, map[string]string, bool) {
	before := len(obj.GetAnnotations()) + len(obj.GetLabels())
	_ = syncerreconcile.RemoveNomosLabelsAndAnnotations(obj)
	core.SetAnnotation(obj, v1.ResourceManagementKey, v1.ResourceManagementDisabled)
	core.RemoveAnnotations(obj, OwningInventoryKey)
	after := len(obj.GetAnnotations()) + len(obj.GetLabels())
	return obj.GetLabels(), obj.GetAnnotations(), before != after
}
