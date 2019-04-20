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
	status.Register(IllegalHierarchyModeErrorCode, IllegalHierarchyModeError{
		HierarchyConfig: fakeHierarchyConfig{
			Resource: hierarchyConfig(),
			gk:       kinds.Role().GroupKind(),
		},
		HierarchyMode: "invalid mode",
		Allowed:       map[v1.HierarchyModeType]bool{v1.HierarchyModeNone: true},
	})
}

// IllegalHierarchyModeError reports that a HierarchyConfig is defined with a disallowed hierarchyMode.
type IllegalHierarchyModeError struct {
	id.HierarchyConfig
	HierarchyMode v1.HierarchyModeType
	Allowed       map[v1.HierarchyModeType]bool
}

var _ status.ResourceError = &IllegalHierarchyModeError{}

// Error implements error
func (e IllegalHierarchyModeError) Error() string {
	var allowedStr []string
	for a := range e.Allowed {
		allowedStr = append(allowedStr, string(a))
	}
	gk := e.GroupKind()
	return status.Format(e,
		"HierarchyMode %q is not a valid value for the APIResource %q. Allowed values are [%s].",
		e.HierarchyMode, gk.String(), strings.Join(allowedStr, ","))
}

// Code implements Error
func (e IllegalHierarchyModeError) Code() string { return IllegalHierarchyModeErrorCode }

// Resources implements ResourceError
func (e IllegalHierarchyModeError) Resources() []id.Resource {
	return []id.Resource{e.HierarchyConfig}
}

// ToCME implements ToCMEr.
func (e IllegalHierarchyModeError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
