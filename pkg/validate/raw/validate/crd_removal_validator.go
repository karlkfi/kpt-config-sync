package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RemovedCRDs verifies that the Raw objects do not remove any CRDs that still
// have CRs using them.
func RemovedCRDs(objs *objects.Raw) status.MultiError {
	current, errs := customresources.GetCRDs(objs.Objects)
	if errs != nil {
		return errs
	}
	removed := removedGroupKinds(objs.PreviousCRDs, current)
	for _, obj := range objs.Objects {
		if removed[obj.GetObjectKind().GroupVersionKind().GroupKind()] {
			errs = status.Append(errs, nonhierarchical.UnsupportedCRDRemovalError(obj))
		}
	}
	return errs
}

type removedGKs map[schema.GroupKind]bool

func removedGroupKinds(previous, current []*v1beta1.CustomResourceDefinition) removedGKs {
	removed := make(map[schema.GroupKind]bool)
	for _, crd := range previous {
		gk := schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
		removed[gk] = true
	}
	for _, crd := range current {
		gk := schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
		delete(removed, gk)
	}
	return removed
}
