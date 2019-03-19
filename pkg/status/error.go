package status

import (
	"fmt"
)

// Error defines a Kubernetes Nomos Vet error
// These are GKE Config Management directory errors which are shown to the user and documented.
type Error interface {
	Error() string
	Code() string
}

// errs is a map from error codes to instances of the types they represent.
// Entries set to nil are reserved and MUST NOT be reused.
var errs = map[string]Error{
	"1023": nil,
	"1025": nil,
}

// Format formats the start of error messages consistently.
func Format(err Error, format string, a ...interface{}) string {
	return fmt.Sprintf("KNV%s: ", err.Code()) + fmt.Sprintf(format, a...)
}

// PathError defines a status error associated with one or more path-identifiable locations in the
// repo.
type PathError interface {
	Error
	RelativePaths() []string
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
