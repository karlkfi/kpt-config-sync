/*
Copyright 2017 The Nomos Authors.
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

package namespaceutil

import (
	"testing"

	"github.com/google/nomos/pkg/api/policyhierarchy"
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
		{"kube-system", true},
		{"kube-public", true},
		{"kube-foo", true},
		{"default", true},
		{policyhierarchy.ControllerNamespace, true},
	} {
		reserved := IsReserved(testcase.name)
		if reserved != testcase.reserved {
			t.Errorf("Expected %v got %v testing %v", testcase.reserved, reserved, testcase)
		}
	}
}
