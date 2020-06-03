package nomostest

import (
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// Retry calls the passed function until it returns nil, or the passed timeout
// expires.
//
// Retries once per second until timeout expires.
func Retry(timeout time.Duration, fn func() error) error {
	return retry.OnError(backoff(timeout), defaultErrorFilter, fn)
}

// backoff returns a wait.Backoff that retries exactly once per second until
// timeout expires.
func backoff(timeout time.Duration) wait.Backoff {
	// These are e2e tests and we aren't doing any sort of load balancing, so
	// for now we don't need to let all aspects of the backoff be configurable.

	// This creates a constant backoff that always retries after exactly one
	// second. See documentation for wait.Backoff for full explanation.
	return wait.Backoff{
		Duration: time.Second,
		Steps:    int(timeout / time.Second),
	}
}

// defaultErrorFilter returns false if the error's type indicates continuing
// will not produce a positive result.
//
// Specifically:
// 1) The wrong type was passed to the predicate.
// 2) The type is not registered in the Client used to talk to the cluster.
func defaultErrorFilter(err error) bool {
	return !errors.Is(err, ErrWrongType) && !runtime.IsNotRegisteredError(err)
}
