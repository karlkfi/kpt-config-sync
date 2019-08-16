package nonhierarchical

import "github.com/google/nomos/pkg/importer/analyzer/validation/metadata"

// ManagementAnnotationValidator ensures the passed object either has no Managment annotation, or
//  declares a valid one.
var ManagementAnnotationValidator = perObjectValidator(metadata.ValidManagementAnnotation)
