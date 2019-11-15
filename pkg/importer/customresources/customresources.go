package customresources

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/gatekeeper"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ProcessClusterObjects will process all the given cluster objects into the
// list of all CRDs that will result
func ProcessClusterObjects(clusterObjects []*ast.ClusterObject) (map[schema.GroupKind]*v1beta1.CustomResourceDefinition, status.MultiError) {
	fileObjects := make([]ast.FileObject, len(clusterObjects))
	for idx := range clusterObjects {
		fileObjects[idx] = clusterObjects[idx].FileObject
	}
	return Process(fileObjects)
}

// Process will process all given objects into the resulting list of CRDs.
// This has special handling for gatekeeper ConstraintTemplates since the
// gatekeeper controller will create a CRD on apply of the ConstraintTemplate.
func Process(fileObjects []ast.FileObject) (map[schema.GroupKind]*v1beta1.CustomResourceDefinition, status.MultiError) {
	var errs status.MultiError
	crdMap := map[schema.GroupKind]*v1beta1.CustomResourceDefinition{}
	for _, cr := range fileObjects {
		if cr.GroupVersionKind() != kinds.CustomResourceDefinition() {
			continue
		}

		crd, err := clusterconfig.AsCRD(cr.Object)
		if err != nil {
			errs = status.Append(errs, status.PathWrapError(err, cr.SlashPath()))
			continue
		}
		gk := schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
		crdMap[gk] = crd
	}

	for _, f := range fileObjects {
		if f.GroupVersionKind().GroupKind() != gatekeeper.ConstraintTemplateGroupKind {
			continue
		}

		version := f.GroupVersionKind().Version
		if version != "v1alpha1" && version != "v1beta1" {
			errs = status.Append(errs, errors.Errorf("unhandled ConstraintTemplate version %s", version))
			continue
		}

		crd, err := gatekeeper.ConstraintTemplateCRD(f.Object)
		if err != nil {
			errs = status.Append(errs, status.PathWrapError(err, f.SlashPath()))
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
	return crdMap, errs
}
