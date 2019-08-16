package nonhierarchical

import "github.com/google/nomos/pkg/importer/analyzer/validation/metadata"

// NameValidator adapts metadata.NameValidator logic for non-hierarchical file structures.
var NameValidator = perObjectValidator(metadata.ValidName)
