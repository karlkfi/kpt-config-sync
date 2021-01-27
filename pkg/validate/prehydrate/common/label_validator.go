package common

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
)

// LabelValidator returns a visitor that verifies that no object has invalid
// labels in the source repository.
func LabelValidator() parsed.ValidatorFunc {
	return parsed.ValidateAllObjects(parsed.PerObjectVisitor(validateLabels))
}

func validateLabels(obj ast.FileObject) status.Error {
	var invalid []string
	for l := range obj.GetLabels() {
		if hasConfigSyncPrefix(l) {
			invalid = append(invalid, l)
		}
	}
	if len(invalid) > 0 {
		return metadata.IllegalLabelDefinitionError(&obj, invalid)
	}
	return nil
}
