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
