package status

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
)

const urlBase = "For more information, see https://cloud.google.com/csp-config-management/docs/errors#knv"

func url(code string) string {
	return urlBase + code
}

func knv(id string) string {
	return fmt.Sprintf("KNV%s", id)
}

func prefix(code string) string {
	return fmt.Sprintf("%s: ", code)
}

// Error defines a Kubernetes Nomos Vet error
// These are GKE Config Management directory errors which are shown to the user and documented.
type Error interface {
	// error is the standard error interface.
	error
	// ToCMEr knows how to convert itself to a ConfigManagementError.
	// ToCME converts the implementor into ConfigManagementError, preserving
	// structured information.
	ToCME() v1.ConfigManagementError
	// Code is the unique identifier of the error to help users find documentation.
	Code() string
}

// errs is a map from error codes to instances of the types they represent.
// Entries set to nil are reserved and MUST NOT be reused.
// The map values are sample error values by types only.  They do not need to
// be properly initialized, but need to be of the same type as the error
// corresponding to the code.
var errs = map[string]Error{
	"1023": nil,
	"1025": nil,
}

// Format formats the start of error messages consistently.
func Format(err Error, format string, a ...interface{}) string {
	var sb strings.Builder
	sb.WriteString(prefix(knv(err.Code())))
	sb.WriteString(fmt.Sprintf(format, a...))

	switch e := err.(type) {
	case ResourceError:
		sb.WriteString("\n\n")
		sb.WriteString(formatResources(e))
	case PathError:
		sb.WriteString("\n\n")
		sb.WriteString(formatPaths(e))
	}

	sb.WriteString("\n\n")
	sb.WriteString(url(err.Code()))
	return sb.String()
}

// PathError defines a status error associated with one or more path-identifiable locations in the
// repo.
type PathError interface {
	Error
	RelativePaths() []id.Path
}

func nextCandidate(code string) (int, error) {
	c, err := strconv.Atoi(code)
	if err != nil {
		return 0, err
	}

	for ; true; c++ {
		if _, found := errs[strconv.Itoa(c)]; found {
			continue
		}
		return c, nil
	}
	panic("unreachable code")
}

// Register marks the passed error code as used. err is a sample value of Error
// for this code.
func Register(code string, err Error) {
	if _, exists := errs[code]; exists {
		if c, err2 := nextCandidate(code); err2 == nil {
			panic(fmt.Errorf("duplicate error code %s: %T, next candidate: %d", code, err, c))
		} else {
			panic(fmt.Errorf("duplicate error code %s: %T", code, err))
		}
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

// toErrorResource converts a Resource into a v1.ErrorResource.
func toErrorResource(r id.Resource) v1.ErrorResource {
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
		Code:         knv(err.Code()),
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
		cme.ErrorResources = append(cme.ErrorResources, toErrorResource(r))
	}
	return cme
}
