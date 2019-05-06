package status

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/pkg/errors"
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

// registered is a map from error codes to instances of the types they represent.
// Entries set to true are reserved and MUST NOT be reused.
var registered = map[string]bool{
	"1023": true,
	"1025": true,
}

// examples is a map of examples of each error type. For documentation purposes, i.e. for use
// in the internal-only nomoserrors command.
var examples = make(map[string][]Error)

// format formats error messages consistently.
//
// err is the underlying error.
// references is an already formatted list of directories or objects. Empty string for no references.
// code is the error's documentation id.
func format(err error, references string, code string) string {
	var sb strings.Builder
	sb.WriteString(prefix(knv(code)))
	sb.WriteString(err.Error())

	if references != "" {
		sb.WriteString("\n\n")
		sb.WriteString(references)
	}

	sb.WriteString("\n\n")
	sb.WriteString(url(code))
	return sb.String()
}

// Format formats the start of error messages consistently.
//
// To be deprecated. Prefer NewErrorBuilder.
func Format(err Error, fmt string, a ...interface{}) string {
	references := ""

	switch e := err.(type) {
	case ResourceError:
		references = formatResources(e.Resources())
	case PathError:
		panic("deprecated for PathErrors; use NewErrorBuilder instead")
	}

	return format(errors.Errorf(fmt, a...), references, err.Code())
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
		if _, found := registered[strconv.Itoa(c)]; found {
			continue
		}
		return c, nil
	}
	panic("unreachable code")
}

// Register registers a unique code for use and examples for it.
//
// Soo to be deprecated. Use NewErrorBuilder and AddExamples instead.
func Register(code string, errs ...Error) {
	register(code)
	AddExamples(code, errs...)
}

// AddExamples adds examples for a specific error code for use in documentation. For example, via
// the internal-only `nomoserrors` command.
func AddExamples(code string, errs ...Error) {
	for _, err := range errs {
		// Ensures example errors can be displayed.
		_ = err.Error()
	}
	examples[code] = append(examples[code], errs...)
}

// Register marks the passed error code as used. err is a sample value of Error
// for this code.
func register(code string) {
	if _, exists := registered[code]; exists {
		if c, err2 := nextCandidate(code); err2 == nil {
			panic(fmt.Errorf("duplicate error code %s, next candidate: %d", code, c))
		} else {
			panic(fmt.Errorf("duplicate error code %s", code))
		}
	}
	registered[code] = true
}

// Registry returns a copy of the error registry.
func Registry() map[string][]Error {
	result := make(map[string][]Error)
	for code, errs := range examples {
		result[code] = append(result[code], errs...)
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
