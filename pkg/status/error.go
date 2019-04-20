package status

import (
	"fmt"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
)

const urlBase = "https://cloud.google.com/csp-config-management/docs/errors#knv"

func url(err Error) string {
	return urlBase + err.Code()
}

func prefixErr(err Error) string {
	return fmt.Sprintf("KNV%s", err.Code())
}

// ToCMEr knows how to serialize itself to a ConfigManagementError.
type ToCMEr interface {
	// ToCME converts the implementor into ConfigManagementError, preserving
	// structured information.
	ToCME() v1.ConfigManagementError
}

// Error defines a Kubernetes Nomos Vet error
// These are GKE Config Management directory errors which are shown to the user and documented.
type Error interface {
	// error is the standard error interface.
	error
	// ToCMEr knows how to convert itself to a ConfigManagementError.
	ToCMEr
	// Code is the unique identifier of the error to help users find documentation.
	Code() string
}

// errs is a map from error codes to instances of the types they represent.
// Entries set to nil are reserved and MUST NOT be reused.
var errs = map[string]Error{
	"1023": nil,
	"1025": nil,
}

// Format formats the start of error messages consistently.
// nolint:errcheck
func Format(err Error, format string, a ...interface{}) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s: ", prefixErr(err))
	fmt.Fprintf(&sb, format, a...)

	switch e := err.(type) {
	case ResourceError:
		sb.WriteString("\n\n")
		sb.WriteString(formatResources(e))
	case PathError:
		sb.WriteString("\n\n")
		sb.WriteString(formatPaths(e))
	}

	fmt.Fprintf(&sb, "\n\nFor more information, see %s", url(err))
	return sb.String()
}

// PathError defines a status error associated with one or more path-identifiable locations in the
// repo.
type PathError interface {
	Error
	RelativePaths() []id.Path
}

// Register marks the passed error code as used.
func Register(code string, err Error) {
	if _, exists := errs[code]; exists {
		panic(fmt.Errorf("duplicate error code %s: %T", code, err))
	}
	errs[code] = err
}

// Registry returns a copy of the error registry.
func Registry() map[string]Error {
	result := make(map[string]Error)
	for code, err := range errs {
		result[code] = err
	}
	return result
}

// ToErrorResource converts a Resource into a v1.ErrorResource.
func ToErrorResource(r id.Resource) v1.ErrorResource {
	return v1.ErrorResource{
		SourcePath:        r.SlashPath(),
		ResourceName:      r.Name(),
		ResourceNamespace: r.Namespace(),
		ResourceGVK:       r.GroupVersionKind(),
	}
}

// FromError embeds the error message and error code into a ConfigManagementError.
func FromError(err Error) v1.ConfigManagementError {
	return v1.ConfigManagementError{
		ErrorMessage: err.Error(),
		Code:         prefixErr(err),
	}
}

// FromPathError converts a PathError to a ConfigManagementError.
func FromPathError(err PathError) v1.ConfigManagementError {
	cme := FromError(err)
	for _, path := range err.RelativePaths() {
		cme.ErrorResources = append(
			cme.ErrorResources,
			v1.ErrorResource{SourcePath: path.SlashPath()})
	}
	return cme
}

// FromResourceError converts a ResourceError to a ConfigManagementError.
func FromResourceError(err ResourceError) v1.ConfigManagementError {
	cme := FromError(err)
	for _, r := range err.Resources() {
		cme.ErrorResources = append(cme.ErrorResources, ToErrorResource(r))
	}
	return cme
}
