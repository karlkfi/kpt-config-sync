package reconcile

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement"
)

func TestIsSystem(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		reserved bool
	}{
		{"default", false},
		{"foo-bar", false},
		{"kube-foo", true},
		{"kube-public", true},
		{"kube-system", true},
		{configmanagement.ControllerNamespace, false},
	} {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			reserved := isSystemNamespace(testcase.name)
			if reserved != testcase.reserved {
				t.Errorf("Expected %v got %v", testcase.reserved, reserved)
			}
		})
	}
}

func TestIsManageableSystem(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		reserved bool
	}{
		{"default", true},
		{"foo-bar", false},
		{"kube-foo", false},
		{"kube-public", true},
		{"kube-system", true},
		{configmanagement.ControllerNamespace, false},
	} {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			reserved := isManageableSystemNamespace(testcase.name)
			if reserved != testcase.reserved {
				t.Errorf("Expected %v got %v", testcase.reserved, reserved)
			}
		})
	}
}
