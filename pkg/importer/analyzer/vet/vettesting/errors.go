package vettesting

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/status"
)

// ExpectErrors adds an error to testing if the expected and actual errors don't match.
// Does not verify the ordering of errors.
func ExpectErrors(expected []string, err error, t *testing.T) {
	t.Helper()
	actual := ErrorCodeMap(err)
	if diff := cmp.Diff(toMap(expected), actual); diff != "" {
		// All of expected, err and diff are needed to debug this error message
		// effectively when it happens.
		t.Fatalf("expected:\n%v\nactual:\n%v\ndiff:\n%v", expected, err, diff)
	}
}

func toMap(codes []string) map[string]int {
	if len(codes) == 0 {
		return nil
	}

	result := make(map[string]int)
	for _, code := range codes {
		result[code] = result[code] + 1
	}
	return result
}

// ErrorCodeMap returns a map from each error code present to the number of times it occurred.
func ErrorCodeMap(err error) map[string]int {
	return toMap(ErrorCodes(err))
}

// ErrorCodes returns the KNV error codes present in the passed error
func ErrorCodes(err error) []string {
	switch e := err.(type) {
	case nil:
		return []string{}
	case status.Error:
		return []string{e.Code()}
	case status.MultiError:
		if e == nil {
			return []string{}
		}
		var result []string
		for _, er := range e.Errors() {
			result = append(result, ErrorCodes(er)...)
		}
		return result
	default:
		// For errors without a specific code
		return []string{UndefinedErrorCode}
	}
}

// UndefinedErrorCode is the code representing an unregistered error. These should be eliminated.
const UndefinedErrorCode = "????"
