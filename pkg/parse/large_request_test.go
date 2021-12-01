package parse

import (
	"fmt"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestIsRequestTooLargeError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "a nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "RequestEntityTooLargeError",
			err:      apierrors.NewRequestEntityTooLargeError("too large"),
			expected: true,
		},
		{
			name:     "ResourceExhaustedError",
			err:      fmt.Errorf("rpc error: code = ResourceExhausted desc = trying to send message larger than max (3136163 vs. 2097152)"),
			expected: true,
		},
		{
			name:     "etcdserver: request is too large",
			err:      fmt.Errorf("etcdserver: request is too large"),
			expected: true,
		},
		{
			name:     "random error",
			err:      fmt.Errorf("ramdom error"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isRequestTooLargeError(tc.err)
			if got != tc.expected {
				t.Errorf("isRequestTooLargeError(%v) got %v, expected %v", tc.err, got, tc.expected)
			}
		})
	}
}
