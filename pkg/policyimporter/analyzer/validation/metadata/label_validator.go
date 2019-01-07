package metadata

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// LabelValidator validates the labels in a ast.FileObject
var LabelValidator = &syntax.FileObjectValidator{
	ValidateFn: func(o ast.FileObject) error {
		found := invalids(o.ToMeta().GetLabels(), noneAllowed)
		if len(found) > 0 {
			return veterrors.IllegalLabelDefinitionError{ResourceID: &o, Labels: found}
		}
		return nil
	},
}

var noneAllowed = map[string]struct{}{}
