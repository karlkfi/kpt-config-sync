package hierarchyconfig

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

var (
	resourceQuotaModes = map[v1.HierarchyModeType]bool{
		v1.HierarchyModeDefault:           true,
		v1.HierarchyModeHierarchicalQuota: true,
		v1.HierarchyModeInherit:           true,
		v1.HierarchyModeNone:              true,
	}
	otherTypesModes = map[v1.HierarchyModeType]bool{
		v1.HierarchyModeDefault: true,
		v1.HierarchyModeInherit: true,
		v1.HierarchyModeNone:    true,
	}
)

// NewInheritanceValidator returns a visitor that validates the inheritance setting
// of all GroupKinds defined across HierarchyConfigs.
func NewInheritanceValidator() ast.Visitor {
	return visitor.NewSystemObjectValidator(func(o *ast.SystemObject) status.MultiError {
		switch h := o.Object.(type) {
		case *v1.HierarchyConfig:
			for _, gkc := range NewFileHierarchyConfig(h, o).flatten() {
				if err := ValidateInheritance(gkc); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ValidateInheritance returns an error if the HierarchyModeType is invalid for the GroupKind in the
// FileGroupKindHierarchyConfig
func ValidateInheritance(config FileGroupKindHierarchyConfig) status.MultiError {
	if config.GroupKind() == kinds.ResourceQuota().GroupKind() {
		return errIfNotAllowed(config, resourceQuotaModes)
	}
	return errIfNotAllowed(config, otherTypesModes)
}

// errIfNotAllowed returns an error if the kindHierarchyConfig has an inheritance mode which is not allowed for that Kind.
func errIfNotAllowed(config FileGroupKindHierarchyConfig, allowed map[v1.HierarchyModeType]bool) status.MultiError {
	if allowed[config.HierarchyMode] {
		return nil
	}
	return IllegalHierarchyModeError(
		config,
		config.HierarchyMode,
		allowed,
	)
}

// IllegalHierarchyModeErrorCode is the error code for IllegalHierarchyModeError
const IllegalHierarchyModeErrorCode = "1042"

var illegalHierarchyModeError = status.NewErrorBuilder(IllegalHierarchyModeErrorCode)

// IllegalHierarchyModeError reports that a HierarchyConfig is defined with a disallowed hierarchyMode.
func IllegalHierarchyModeError(
	config id.HierarchyConfig,
	mode v1.HierarchyModeType,
	allowed map[v1.HierarchyModeType]bool) status.Error {
	var allowedStr []string
	for a := range allowed {
		allowedStr = append(allowedStr, string(a))
	}
	gk := config.GroupKind()
	return illegalHierarchyModeError.Sprintf(
		"HierarchyMode %q is not a valid value for the APIResource %q. Allowed values are [%s].",
		mode, gk.String(), strings.Join(allowedStr, ",")).BuildWithResources(config)
}
