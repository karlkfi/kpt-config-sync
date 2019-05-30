package vet

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalAbstractNamespaceObjectKindErrorCode is the error code for IllegalAbstractNamespaceObjectKindError
const IllegalAbstractNamespaceObjectKindErrorCode = "1007"

func init() {
	status.AddExamples(IllegalAbstractNamespaceObjectKindErrorCode, IllegalAbstractNamespaceObjectKindError(role()))
}

var illegalAbstractNamespaceObjectKindError = status.NewErrorBuilder(IllegalAbstractNamespaceObjectKindErrorCode)

// IllegalAbstractNamespaceObjectKindError represents an illegal usage of a kind not allowed in abstract namespaces.
// TODO(willbeason): Consolidate Illegal{X}ObjectKindErrors
func IllegalAbstractNamespaceObjectKindError(resource id.Resource) status.Error {
	return illegalAbstractNamespaceObjectKindError.Errorf(
		"Config `%[3]s` illegally declared in an %[1]s directory. "+
			"Move this config to a %[2]s directory:",
		strings.ToLower(string(node.AbstractNamespace)), node.Namespace, resource.Name())
}
