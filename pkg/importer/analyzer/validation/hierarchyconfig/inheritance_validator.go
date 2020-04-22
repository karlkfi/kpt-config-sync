package hierarchyconfig

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// NewInheritanceValidator returns a visitor that validates the inheritance setting
// of all GroupKinds defined across HierarchyConfigs.
func NewInheritanceValidator() ast.Visitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) status.MultiError {
		switch h := o.Object.(type) {
		case *v1.HierarchyConfig:
			for _, gkc := range newFileHierarchyConfig(h, o).flatten() {
				if err := validateInheritance(gkc); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ValidateInheritance returns an error if the HierarchyModeType is invalid for the GroupKind in the
// FileGroupKindHierarchyConfig
func validateInheritance(config FileGroupKindHierarchyConfig) status.MultiError {
	return errIfNotAllowed(config)
}

// errIfNotAllowed returns an error if the kindHierarchyConfig has an inheritance mode which is not allowed for that Kind.
func errIfNotAllowed(config FileGroupKindHierarchyConfig) status.MultiError {
	switch config.HierarchyMode {
	case v1.HierarchyModeNone:
	case v1.HierarchyModeInherit:
	case v1.HierarchyModeDefault:
	default:
		return IllegalHierarchyModeError(config, config.HierarchyMode)
	}

	return nil
}

// IllegalHierarchyModeErrorCode is the error code for IllegalHierarchyModeError
const IllegalHierarchyModeErrorCode = "1042"

var illegalHierarchyModeError = status.NewErrorBuilder(IllegalHierarchyModeErrorCode)

// IllegalHierarchyModeError reports that a HierarchyConfig is defined with a disallowed hierarchyMode.
func IllegalHierarchyModeError(
	config id.HierarchyConfig,
	mode v1.HierarchyModeType) status.Error {
	allowedStr := []string{string(v1.HierarchyModeNone), string(v1.HierarchyModeInherit)}
	gk := config.GroupKind()
	return illegalHierarchyModeError.Sprintf(
		"HierarchyMode %q is not a valid value for the APIResource %q. Allowed values are [%s].",
		mode, gk.String(), strings.Join(allowedStr, ",")).BuildWithResources(config)
}
