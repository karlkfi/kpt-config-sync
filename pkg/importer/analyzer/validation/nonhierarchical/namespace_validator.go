package nonhierarchical

import (
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/status"
)

// NamespaceValidator validates the metadata.namespace field on resources.
var NamespaceValidator = perObjectValidator(validNamespace)

func validNamespace(o ast.FileObject) status.Error {
	if o.GetNamespace() == "" {
		return nil
	}

	// Do note that IsDNS1123Label is misleading. It refers to RFC 1123, and additionally forbids
	// capital letters.
	errs := validation.IsDNS1123Label(o.GetNamespace())
	if errs != nil {
		return vet.InvalidNamespaceError(&o, errs)
	}
	return nil
}
