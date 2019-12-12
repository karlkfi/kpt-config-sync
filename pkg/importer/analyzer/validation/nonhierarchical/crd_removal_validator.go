package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRDRemovalValidator ensures the repo doesn't declare resources which use now-nonexistent CRDs.
//
// syncedCRDs is the set of CRDs currently synced to the cluster from the CRDClusterConfig.
// declaredCRDs is the set of CRDs currently declared in the repository.
//
// A CRD is "pending removal" if it is synced but no longer declared, as this means
// the CRD has been removed since the repo was last synced.
func CRDRemovalValidator(syncedCRDs []*v1beta1.CustomResourceDefinition, declaredCRDs []*v1beta1.CustomResourceDefinition) Validator {
	pendingRemoval := make(map[schema.GroupKind]bool)
	for _, synced := range syncedCRDs {
		pendingRemoval[groupKind(synced)] = true
	}
	for _, declared := range declaredCRDs {
		pendingRemoval[groupKind(declared)] = false
	}

	return PerObjectValidator(func(o ast.FileObject) status.Error {
		return checkCRDPendingRemoval(pendingRemoval, o)
	})
}

// groupKind returns the GroupKind of the type the CRD declares.
func groupKind(crd *v1beta1.CustomResourceDefinition) schema.GroupKind {
	return schema.GroupKind{Group: crd.Spec.Group, Kind: crd.Spec.Names.Kind}
}

// checkCRDPendingRemoval returns an error if the type is from a CRD pending removal.
func checkCRDPendingRemoval(pendingRemoval map[schema.GroupKind]bool, o ast.FileObject) status.Error {
	isPendingRemoval, isKnown := pendingRemoval[o.GroupVersionKind().GroupKind()]
	if isKnown {
		// If the type isn't synced or declared, then it is either on the APIServer or we've shown an error already.
		if isPendingRemoval {
			return UnsupportedCRDRemovalError(o)
		}
	}
	return nil
}

// UnsupportedCRDRemovalErrorCode is the error code for UnsupportedCRDRemovalError
const UnsupportedCRDRemovalErrorCode = "1047"

var unsupportedCRDRemovalError = status.NewErrorBuilder(UnsupportedCRDRemovalErrorCode)

// UnsupportedCRDRemovalError reports than a CRD was removed, but its corresponding CRs weren't.
func UnsupportedCRDRemovalError(resource id.Resource) status.Error {
	return unsupportedCRDRemovalError.
		Sprintf("Custom Resources MUST be removed in the same commit as their corresponding " +
			"CustomResourceDefinition. To fix, remove this Custom Resource or re-add the CRD.").
		BuildWithResources(resource)
}
