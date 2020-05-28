package status

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

// MultiError represents a collection of errors.
type MultiError interface {
	error
	Errors() []Error
}

// Append adds one or more errors to an existing MultiError.
// If m, err, and errs are nil, returns nil.
//
// Requires at least one error to be passed explicitly to prevent developer mistakes.
// There is no valid reason to call Append with exactly one argument.
//
// If err is a MultiError, appends all contained errors.
func Append(m MultiError, err error, errs ...error) MultiError {
	result := &multiError{}

	switch m.(type) {
	case nil:
		// No errors to begin with.
	case *multiError:
		result.errs = m.Errors()
	default:
		for _, e := range m.Errors() {
			result.add(e)
		}
	}

	result.add(err)
	for _, e := range errs {
		result.add(e)
	}

	if len(result.errs) == 0 {
		return nil
	}
	return result
}

// ToCME converts a MultiError to ConfigManagementError.
func ToCME(m MultiError) []v1.ConfigManagementError {
	var cmes []v1.ConfigManagementError

	if m != nil {
		for _, err := range m.Errors() {
			cmes = append(cmes, err.ToCME())
		}
	}

	return cmes
}

var _ MultiError = (*multiError)(nil)

// MultiError is an error that contains multiple errors.
type multiError struct {
	errs []Error
}

// Add adds error to the builder.
// If the type is known to contain an array of error, adds all of the contained errors.
// If the error is nil, do nothing.
func (m *multiError) add(err error) {
	switch e := err.(type) {
	case nil:
		// No error to add if nil.
	case Error:
		m.errs = append(m.errs, e)
	case MultiError:
		m.errs = append(m.errs, e.Errors()...)
	case utilerrors.Aggregate:
		for _, er := range e.Errors() {
			m.add(er)
		}
	default:
		m.errs = append(m.errs, undocumented(err))
	}
}

// Error implements error
func (m *multiError) Error() string {
	return FormatError(true, m)
}

// Errors returns a list of the contained errors
func (m *multiError) Errors() []Error {
	if m == nil {
		return nil
	}
	return m.errs
}

// FormatError formats the multiple errors using multiline argument.
// When multiline set to true, errors are formatted and joined using new lines.
// Else, multiple errors are joined using comma separator.
// Sample formatted errors: https://paste.googleplex.com/6533732804591616
// TODO(b/158022901) Sctructure multiline error messages
func FormatError(multiline bool, e error) string {
	m := toMultiError(e)

	// mErrs is a slice of Error.
	mErrs := m.Errors()

	if len(mErrs) == 0 {
		return ""
	}

	var msgs []string
	for _, err := range mErrs {
		msgs = append(msgs, err.Error())
	}
	sort.Strings(msgs)

	// Since errors are sorted by message we can eliminate duplicates by comparing the current
	// error message with the previous.
	var uniqueErrors = make([]string, 0)
	for idx, err := range msgs {
		if idx == 0 || msgs[idx-1] != err {
			uniqueErrors = append(uniqueErrors, err)
		}
	}

	allErrors := []string{
		fmt.Sprintf("%d error(s)\n", len(uniqueErrors)),
	}
	for idx, err := range uniqueErrors {
		allErrors = append(allErrors, fmt.Sprintf("[%d] %v\n", idx+1, err))
	}
	// if in single-line mode, remove new lines from each error message,
	// return error messages joined with a new line.
	if !multiline {
		for idx, err := range allErrors {
			// removes all the new lines from the errors.
			allErrors[idx] = rmNewlines(err)
		}
		return strings.Join(allErrors, "\n")
	}
	return strings.Join(allErrors, "\n\n")
}

func rmNewlines(err string) string {
	return strings.ReplaceAll(err, "\n", " ")
}

func toMultiError(e error) MultiError {
	if me, ok := e.(MultiError); ok {
		return me
	}
	var m MultiError
	return Append(m, e)
}
