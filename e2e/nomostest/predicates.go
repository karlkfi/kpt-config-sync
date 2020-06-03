package nomostest

import (
	"fmt"

	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
)

// Predicate evaluates a core.Object, returning an error if it fails validation.
type Predicate func(o core.Object) error

// ErrWrongType indicates that the caller passed an object of the incorrect type
// to the Predicate.
var ErrWrongType = errors.New("wrong type")

// WrongTypeErr reports that the passed type was not equivalent to the wanted
// type.
func WrongTypeErr(got, want interface{}) error {
	return fmt.Errorf("%w: got %T, want %T", ErrWrongType, got, want)
}

// ErrFailedPredicate indicates the the object on the API server does not match
// the Predicate.
var ErrFailedPredicate = errors.New("failed predicate")
