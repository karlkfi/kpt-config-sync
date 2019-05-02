/*
Copyright 2018 The CSP Config Management Authors.
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

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

// MultiError represents a collection of errors.
type MultiError interface {
	error
	Errors() []Error
}

// From creates a MultiError from one or more errors.
// If err is nil, returns nil.
func From(errs ...error) MultiError {
	return Append(nil, errs...)
}

// Append adds one or more errors to an existing MultiError.
// If m and err are nil, returns nil.
func Append(m MultiError, errs ...error) MultiError {
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
	case utilerrors.Aggregate:
		for _, er := range e.Errors() {
			m.add(er)
		}
	case MultiError:
		m.errs = append(m.errs, e.Errors()...)
	default:
		m.errs = append(m.errs, UndocumentedWrapf(err, ""))
	}
}

// Error implements error
func (m *multiError) Error() string {
	if m == nil {
		return ""
	}

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
func (m *multiError) Errors() []Error {
	if m == nil {
		return nil
	}
	return m.errs
}
