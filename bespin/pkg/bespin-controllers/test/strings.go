package test

import (
	"fmt"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/uuid"
)

// Name returns a suitable name for an object based on the test name.
func Name(t *testing.T) string {
	return strings.ToLower(strings.TrimPrefix(t.Name(), "TestReconcile"))
}

// UniqueName takes a name and returns a unique version.
func UniqueName(t *testing.T, name string) string {
	return fmt.Sprintf("%v-%v", name, uuid.NewUUID())
}
