package customresources

import (
	"sort"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetCRDs will process all given objects into the resulting list of CRDs.
func GetCRDs(fileObjects []ast.FileObject) ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
	var errs status.MultiError
	crdMap := map[schema.GroupKind]*v1beta1.CustomResourceDefinition{}
	for _, obj := range fileObjects {
		if obj.GetObjectKind().GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
			continue
		}

		crd, err := clusterconfig.AsCRD(obj.Unstructured)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		gk := schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
		crdMap[gk] = crd
	}

	var result []*v1beta1.CustomResourceDefinition
	for _, crd := range crdMap {
		result = append(result, crd)
	}
	// Sort to ensure deterministic list order.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, errs
}
