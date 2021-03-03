package validate

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/util/clusterconfig"
)

var illegalGroups = map[string]bool{
	v1.SchemeGroupVersion.Group:       true,
	v1alpha1.SchemeGroupVersion.Group: true,
}

// IllegalCRD returns an error if the given FileObject is a CRD of a Config Sync
// type.
func IllegalCRD(obj ast.FileObject) status.Error {
	if obj.GroupVersionKind().GroupKind() != kinds.CustomResourceDefinition() {
		return nil
	}

	crd, err := clusterconfig.AsCRD(obj.Object)
	if err != nil {
		return err
	}
	if illegalGroups[crd.Spec.Group] {
		return nonhierarchical.UnsupportedObjectError(obj)
	}
	return nil
}
