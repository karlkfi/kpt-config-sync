package filesystem

import (
	"github.com/google/nomos/pkg/importer/customresources"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// readCRDs returns the list of CRDs in a directory.
func readCRDs(r Reader, dir cmpath.Relative) ([]*v1beta1.CustomResourceDefinition, status.MultiError) {
	fileObjects, errs := r.Read(dir, true, nil)
	if errs != nil {
		return nil, errs
	}

	return customresources.GetCRDs(fileObjects)
}
