/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package status

import (
	"fmt"
	"sort"
	"strings"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

// ErrorBuilder builds MultiErrors. Instantiate directly as:
//
//     b := &status.ErrorBuilder{}
type ErrorBuilder struct {
	errs []Error
}

// from returns a MultiError with the array of errors.
func from(errs []error) MultiError {
	b := ErrorBuilder{}
	for _, err := range errs {
		b.Add(err)
	}
	result := b.Build()
	if result == nil {
		return MultiError{}
	}
	return *result.(*MultiError)
}

// Add adds error to the builder.
// If the type is known to contain an array of error, adds all of the contained errors.
// If the error is nil, do nothing.
func (b *ErrorBuilder) Add(err error) {
	switch e := err.(type) {
	case nil:
		// No error to add if nil.
	case Error:
		b.errs = append(b.errs, e)
	case utilerrors.Aggregate:
		b.Add(from(e.Errors()))
	case MultiError:
		b.errs = append(b.errs, e.errs...)
	case *MultiError:
		b.errs = append(b.errs, e.errs...)
	default:
		b.Add(UndocumentedWrapf(err, ""))
	}
}

// Build builds the error or returns nil if no errors were added
func (b *ErrorBuilder) Build() error {
	if len(b.errs) == 0 {
		return nil
	}
	return &MultiError{errs: b.errs}
}

// Len returns the number of errors in the builder.
func (b *ErrorBuilder) Len() int {
	return len(b.errs)
}

// HasErrors returns true if there are errors in the builder.
func (b *ErrorBuilder) HasErrors() bool {
	return b.Len() > 0
}

func (b *ErrorBuilder) Error() string {
	return b.Build().Error()
}

// MultiError is an error that contains multiple errors.
type MultiError struct {
	errs []Error
}

// Error implements error
func (m MultiError) Error() string {
	// sort errors alphabetically by their message.
	sort.Slice(m.errs, func(i, j int) bool {
		return m.errs[i].Error() < m.errs[j].Error()
	})

	// Since errors are sorted by message we can eliminate duplicates by comparing the current
	// error message with the previous.
	var uniqueErrors = make([]error, 0)
	for idx, err := range m.errs {
		if idx == 0 || m.errs[idx-1].Error() != err.Error() {
			uniqueErrors = append(uniqueErrors, err)
		}
	}

	allErrors := []string{
		fmt.Sprintf("%d error(s)\n", len(uniqueErrors)),
	}
	for idx, err := range uniqueErrors {
		allErrors = append(allErrors, fmt.Sprintf("[%d] %v\n", idx+1, err))
	}
	return strings.Join(allErrors, "\n\n")
}

// Errors returns a list of the contained errors
func (m MultiError) Errors() []Error {
	return m.errs
}
