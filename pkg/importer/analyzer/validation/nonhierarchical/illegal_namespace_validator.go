package nonhierarchical

import (
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// IllegalNamespaceErrorCode is the error code for illegal Namespace definitions.
const IllegalNamespaceErrorCode = "1034"

var illegalNamespaceError = status.NewErrorBuilder(IllegalNamespaceErrorCode)

func objectInIllegalNamespace(resource id.Resource) status.Error {
	return illegalNamespaceError.WithResources(resource).Errorf(
		"Configs must not be declared in the %q namespace", configmanagement.ControllerNamespace,
	)
}

func illegalNamespace(resource id.Resource) status.Error {
	return illegalNamespaceError.WithResources(resource).Errorf(
		"The %q Namespace must not be declared", configmanagement.ControllerNamespace,
	)
}

// IllegalNamespaceValidator forbids declaring resources in the ControllerNamespace.
var IllegalNamespaceValidator = perObjectValidator(func(object ast.FileObject) status.Error {
	if object.GetNamespace() == configmanagement.ControllerNamespace {
		return objectInIllegalNamespace(&object)
	}
	if object.GroupVersionKind() == kinds.Namespace() && object.GetName() == configmanagement.ControllerNamespace {
		return illegalNamespace(&object)
	}
	return nil
})
