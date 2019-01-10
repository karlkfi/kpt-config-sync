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

package test

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/client/action"
)

// Action implements action.Interface for testing
type Action struct {
	namespace string
	name      string
	resource  string
	operation action.OperationType
}

// Operation implements action.Interface
func (s *Action) Operation() action.OperationType {
	return s.operation
}

// Execute implements action.Interface
func (s *Action) Execute() error {
	return nil
}

// Resource implements action.Interface
func (s *Action) Resource() string {
	return s.resource
}

// Kind implements action.Interface
func (s *Action) Kind() string {
	return strings.Title(s.resource)
}

// Namespace implements action.Interface
func (s *Action) Namespace() string {
	return s.namespace
}

// Group implements action.Interface
func (s *Action) Group() string {
	return "group"
}

// Version implements action.Interface
func (s *Action) Version() string {
	return "v1"
}

// Name implements action.Interface
func (s *Action) Name() string {
	return s.name
}

// String implements action.Interface
func (s *Action) String() string {
	if ns := s.Namespace(); ns != "" {
		return fmt.Sprintf(
			"%s/%s/%s/%s/%s/%s",
			s.Group(),
			s.Version(),
			s.Kind(),
			ns,
			s.Name(),
			s.Operation())
	}
	return fmt.Sprintf(
		"%s/%s/%s/%s/%s",
		s.Group(),
		s.Version(),
		s.Kind(),
		s.Name(),
		s.Operation())
}

// NewDelete creates a new test delete action
func NewDelete(namespace, name, resource string) action.Interface {
	return &Action{
		namespace: namespace,
		name:      name,
		resource:  resource,
		operation: action.DeleteOperation,
	}
}

// NewUpsert creates a new test upsert action
func NewUpsert(namespace, name, resource string) action.Interface {
	return &Action{
		namespace: namespace,
		name:      name,
		resource:  resource,
		operation: action.UpsertOperation,
	}
}
