package namespaceutil

import (
	"testing"

	"github.com/google/nomos/pkg/api/configmanagement"
)

type reservedOrInvalidNamespaceTestcase struct {
	name    string
	invalid bool
}

func TestIsInvalid(t *testing.T) {
	for _, testcase := range []reservedOrInvalidNamespaceTestcase{
		{"foo-bar", true},
		{"Foo-Bar", false},
		{"Foo_Bar", false},
		{"-Foo_Bar", false},
		{"Foo_Bar-", false},
		{"ALL-CAPS", false},
		{"-foo-bar", false},
		{"foo-bar-", false},
	} {
		if IsInvalid(testcase.name) && testcase.invalid {
			t.Errorf("Expected error but didn't get one, testing %q", testcase.name)
		}

		if !IsInvalid(testcase.name) && !testcase.invalid {
			t.Errorf("Unexpected testing %q", testcase.name)
		}
	}
}

func TestIsReserved(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		reserved bool
	}{
		{"foo-bar", false},
		{"kube-system", false},
		{"kube-public", false},
		{"kube-foo", false},
		{"default", false},
		{configmanagement.ControllerNamespace, true},
	} {
		testcase := testcase
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			reserved := IsReserved(testcase.name)
			if reserved != testcase.reserved {
				t.Errorf("Expected %v got %v", testcase.reserved, reserved)
			}
		})
	}
}

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
			reserved := IsSystem(testcase.name)
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
			reserved := IsManageableSystem(testcase.name)
			if reserved != testcase.reserved {
				t.Errorf("Expected %v got %v", testcase.reserved, reserved)
			}
		})
	}
}
