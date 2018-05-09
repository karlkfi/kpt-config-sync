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

package multierror

import (
	"fmt"
	"strings"
)

// Builder builds MultiErrors
type Builder struct {
	errs []error
}

// Add adds an error to the builder
func (b *Builder) Add(err error) {
	b.errs = append(b.errs, err)
}

// Build builds the error or returns nil if no errors were added
func (b *Builder) Build() error {
	if len(b.errs) == 0 {
		return nil
	}
	return &MultiError{errs: b.errs}
}

// Len returns the number of errors in the builder.
func (b *Builder) Len() int {
	return len(b.errs)
}

// NewBuilder returns a MultiError builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// MultiError is an error that contains multiple errors.
type MultiError struct {
	errs []error
}

// Error implements error
func (m *MultiError) Error() string {
	allErrors := []string{
		fmt.Sprintf("Error Count: %d", len(m.errs)),
	}
	for idx, err := range m.errs {
		allErrors = append(allErrors, fmt.Sprintf("error %d:\n%v", idx+1, err))
	}
	return strings.Join(allErrors, "\n")
}
