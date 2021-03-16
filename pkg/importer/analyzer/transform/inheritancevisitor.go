package transform

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
)

// InheritanceSpec defines the spec for inherited resources.
type InheritanceSpec struct {
	Mode v1.HierarchyModeType
}
