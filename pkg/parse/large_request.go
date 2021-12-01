package parse

import (
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const defaultDenominator = 1

// isRequestTooLargeError determines whether `err` was caused by a large request.
//
// References:
//  1) https://github.com/kubernetes/kubernetes/issues/74600
//  2) https://github.com/kubernetes/kubernetes/blob/b0bc8adbc2178e15872f9ef040355c51c45d04bb/test/integration/controlplane/synthetic_controlplane_test.go#L310
func isRequestTooLargeError(err error) bool {
	if err == nil {
		return false
	}

	// apierrors.IsRequestEntityTooLargeError(err) is true if the request size is over 3MB
	if apierrors.IsRequestEntityTooLargeError(err) {
		return true
	}

	// the error message includes `rpc error: code = ResourceExhausted desc = trying to send message larger than max` if the request size is over 2MB
	expectedMsgFor2MB := `rpc error: code = ResourceExhausted desc = trying to send message larger than max`
	if strings.Contains(err.Error(), expectedMsgFor2MB) {
		return true
	}

	// the error message includes `etcdserver: request is too large` if the request size is over 1MB
	expectedMsgFor1MB := `etcdserver: request is too large`
	return strings.Contains(err.Error(), expectedMsgFor1MB)
}
