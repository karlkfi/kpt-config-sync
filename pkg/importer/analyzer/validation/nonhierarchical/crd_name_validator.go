package nonhierarchical

import "github.com/google/nomos/pkg/importer/analyzer/validation/syntax"

// CRDNameValidator validates that CRDs have the expected metadata.name.
var CRDNameValidator = perObjectValidator(syntax.ValidateCRDName)
