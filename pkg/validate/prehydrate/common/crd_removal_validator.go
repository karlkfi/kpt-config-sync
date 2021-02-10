package common

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRDRemovalValidator returns a visitor that verifies that the Root does not
// contain any CRs whose CRD is being removed.
func CRDRemovalValidator(previous, current []*v1beta1.CustomResourceDefinition) parsed.ValidatorFunc {
	removed := removedGroupKinds(previous, current)
	return func(root parsed.Root) status.MultiError {
		return root.VisitAllObjects(parsed.PerObjectVisitor(removed.check))
	}
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

func (r removedGKs) check(obj ast.FileObject) status.Error {
	if r[obj.GroupVersionKind().GroupKind()] {
		return nonhierarchical.UnsupportedCRDRemovalError(obj)
	}
	return nil
}
