package gcp_test

import (
	"fmt"
	"testing"

	"github.com/google/nomos/bespin/pkg/controllers/test/gcp"
	"google.golang.org/api/googleapi"
)

func TestIsNotFoundError(t *testing.T) {
	testCases := []struct {
		Name           string
		Error          error
		ExpectedResult bool
	}{
		{"GCP NotFound", &googleapi.Error{Code: 404}, true},
		{"GCP NotAuthorized", &googleapi.Error{Code: 403}, false},
		{"Generic", fmt.Errorf("my generic error"), false},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := gcp.IsNotFoundError(tc.Error)
			if result != tc.ExpectedResult {
				t.Errorf("unexpected result for gcp.IsNotFoundError('%v'): got '%v', want '%v'", tc.Error, result, tc.ExpectedResult)
			}
		})
	}
}

func TestIsNotAuthorizedError(t *testing.T) {
	testCases := []struct {
		Name           string
		Error          error
		ExpectedResult bool
	}{
		{"GCP NotAuthorized", &googleapi.Error{Code: 403}, true},
		{"GCP NotFound", &googleapi.Error{Code: 404}, false},
		{"Generic", fmt.Errorf("my generic error"), false},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := gcp.IsNotAuthorizedError(tc.Error)
			if result != tc.ExpectedResult {
				t.Errorf("unexpected result for gcp.IsNotAuthorizedError('%v'): got '%v', want '%v'", tc.Error, result, tc.ExpectedResult)
			}
		})
	}
}
