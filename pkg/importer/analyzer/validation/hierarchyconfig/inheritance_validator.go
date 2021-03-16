package hierarchyconfig

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

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
