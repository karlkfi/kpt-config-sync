package vet

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalAnnotationDefinitionErrorCode is the error code for IllegalAnnotationDefinitionError
const IllegalAnnotationDefinitionErrorCode = "1010"

func init() {
	status.Register(IllegalAnnotationDefinitionErrorCode, IllegalAnnotationDefinitionError{
		Resource:    role(),
		Annotations: []string{v1.ResourceManagementKey, v1.SyncTokenAnnotationKey},
	})
}

// IllegalAnnotationDefinitionError represents a set of illegal annotation definitions.
type IllegalAnnotationDefinitionError struct {
	id.Resource
	Annotations []string
}

var _ status.ResourceError = IllegalAnnotationDefinitionError{}

// Error implements error.
func (e IllegalAnnotationDefinitionError) Error() string {
	annotations := e.Annotations
	sort.Strings(annotations) // ensure deterministic annotation order
	annotations2 := make([]string, len(annotations))
	for i, annotation := range annotations {
		annotations2[i] = fmt.Sprintf("%q", annotation)
	}
	a := strings.Join(annotations2, ", ")
	return status.Format(e,
		"Configs MUST NOT declare unsupported annotations starting with %q. "+
			"The config has invalid annotations: %s",
		v1.ConfigManagementPrefix, a)
}

// Code implements Error
func (e IllegalAnnotationDefinitionError) Code() string { return IllegalAnnotationDefinitionErrorCode }

// Resources implements ResourceError
func (e IllegalAnnotationDefinitionError) Resources() []id.Resource {
	return []id.Resource{e.Resource}
}

// ToCME implements ToCMEr.
func (e IllegalAnnotationDefinitionError) ToCME() v1.ConfigManagementError {
	return status.FromResourceError(e)
}
