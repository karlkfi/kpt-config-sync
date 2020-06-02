package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalHierarchicalKindErrorCode is the error code for illegalHierarchicalKindErrors.
const IllegalHierarchicalKindErrorCode = "1032"

var illegalHierarchicalKindError = status.NewErrorBuilder(IllegalHierarchicalKindErrorCode)

// IllegalHierarchicalKind reports that a type is not permitted if hierarchical parsing is disabled.
func IllegalHierarchicalKind(resource id.Resource) status.Error {
	return illegalHierarchicalKindError.
		Sprintf("The type %v is not allowed if `sourceFormat` is set to "+
			"`unstructured`. To fix, remove the problematic config, or convert your repo "+
			"to use `sourceFormat: hierarchy`.", resource.GroupVersionKind().GroupKind().String()).
		BuildWithResources(resource)
}

// IllegalHierarchicalKindValidator forbids declaring configmanagement kinds.
//
// The Nomos Hierarchy has been disabled, using any Nomos type is illegal.
var IllegalHierarchicalKindValidator = PerObjectValidator(func(object ast.FileObject) status.Error {
	if object.GroupVersionKind().Group == configmanagement.GroupName {
		return IllegalHierarchicalKind(&object)
	}
	return nil
})
