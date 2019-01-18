package gcp

import (
	"net/http"

	"google.golang.org/api/googleapi"
)

// IsNotAuthorizedError returns whether the call isn't authorized.
func IsNotAuthorizedError(err error) bool {
	return isGoogleErrorWithCode(err, http.StatusForbidden)
}

// IsNotFoundError returns whether the call refers to a resource that doesn't exist.
func IsNotFoundError(err error) bool {
	return isGoogleErrorWithCode(err, http.StatusNotFound)
}

func isGoogleErrorWithCode(err error, code int) bool {
	if err == nil {
		return false
	}
	if ge, ok := err.(*googleapi.Error); ok {
		return ge.Code == code
	}
	return false
}
