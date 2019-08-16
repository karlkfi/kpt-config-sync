package nonhierarchical

import (
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
)

// IllegalCRDValidator forbids CRDs declaring Nomos types.
var IllegalCRDValidator = perObjectValidator(syntax.IllegalCRD)
