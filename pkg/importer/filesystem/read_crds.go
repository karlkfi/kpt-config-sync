package filesystem

import (
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// readCRDs returns the list of CRDs in a directory.
func readCRDs(r Reader, dir cmpath.Relative) ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
	fileObjects, errs := r.Read(dir, true)
	if errs != nil {
		return nil, errs
	}

	var crds []*v1beta1.CustomResourceDefinition
	for _, f := range fileObjects {
		// Rely on type meta instead of casting as the CRD could be an Unstructured.
		if f.GroupVersionKind() != kinds.CustomResourceDefinition() {
			continue
		}

		crd, err := clusterconfig.AsCRD(f.Object)
		if err != nil {
			// Collect all CRD conversion errors together.
			errs = status.Append(errs, err)
			continue
		}
		crds = append(crds, crd)
	}
	return crds, errs
}
