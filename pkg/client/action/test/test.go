/*
Copyright 2018 The Stolos Authors.
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

package test

import (
	"fmt"

	"github.com/google/nomos/pkg/client/action"
)

// TestAction implements action.Interface for testing
type TestAction struct {
	namespace string
	name      string
	resource  string
	operation string
}

// Operation implements action.Interface
func (s *TestAction) Operation() string {
	return s.operation
}

// Execute implements action.Interface
func (s *TestAction) Execute() error {
	return nil
}

// Resource implements action.Interface
func (s *TestAction) Resource() string {
	return s.operation
}

// Namespace implements action.Interface
func (s *TestAction) Namespace() string {
	return s.operation
}

// String implements action.Interface
func (s *TestAction) String() string {
	if s.namespace == "" {
		return fmt.Sprintf("%s.%s.%s", s.resource, s.name, s.operation)
	}
	return fmt.Sprintf("%s.%s.%s.%s", s.resource, s.namespace, s.name, s.operation)
}

// NewDelete creates a new test delete action
func NewDelete(namespace, name, resource string) action.Interface {
	return &TestAction{
		namespace: namespace,
		name:      name,
		resource:  resource,
		operation: string(action.DeleteOperation),
	}
}

// NewUpsert creates a new test upsert action
func NewUpsert(namespace, name, resource string) action.Interface {
	return &TestAction{
		namespace: namespace,
		name:      name,
		resource:  resource,
		operation: string(action.UpsertOperation),
	}
}
