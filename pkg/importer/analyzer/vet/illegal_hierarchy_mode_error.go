package vet

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// IllegalHierarchyModeErrorCode is the error code for IllegalHierarchyModeError
const IllegalHierarchyModeErrorCode = "1042"

func init() {
	status.AddExamples(IllegalHierarchyModeErrorCode, IllegalHierarchyModeError(
		fakeHierarchyConfig{
			Resource: hierarchyConfig(),
			gk:       kinds.Role().GroupKind(),
		},
		"invalid mode",
		map[v1.HierarchyModeType]bool{v1.HierarchyModeNone: true},
	))
}

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
	return illegalHierarchyModeError.WithResources(config).Errorf(
		"HierarchyMode %q is not a valid value for the APIResource %q. Allowed values are [%s].",
		mode, gk.String(), strings.Join(allowedStr, ","))
}
