package customresources

import (
	"sort"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/gatekeeper"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetCRDs will process all given objects into the resulting list of CRDs.
// This has special handling for gatekeeper ConstraintTemplates since the
// gatekeeper controller will create a CRD on apply of the ConstraintTemplate.
func GetCRDs(fileObjects []ast.FileObject) ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
	var errs status.MultiError
	crdMap := map[schema.GroupKind]*v1beta1.CustomResourceDefinition{}
	for _, obj := range fileObjects {
		if obj.GetObjectKind().GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
			continue
		}

		crd, err := clusterconfig.AsCRD(obj.Object)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		gk := schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
		crdMap[gk] = crd
	}

	for _, f := range fileObjects {
		if f.GetObjectKind().GroupVersionKind().GroupKind() != gatekeeper.ConstraintTemplateGroupKind {
			continue
		}

		version := f.GetObjectKind().GroupVersionKind().Version
		if version != "v1alpha1" && version != "v1beta1" {
			errs = status.Append(errs, errors.Errorf("unhandled ConstraintTemplate version %s", version))
			continue
		}

		crd, err := gatekeeper.ConstraintTemplateCRD(f.Object)
		if err != nil {
			errs = status.Append(errs, clusterconfig.MalformedCRDError(err, f.Object))
			continue
		}

		gk := schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
		if _, found := crdMap[gk]; found {
			// For some reason someone put a gatekeeper generated CRD in the repo,
			// rely on that instead.
			continue
		}
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
