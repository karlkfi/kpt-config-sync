/*
Copyright 2017 The Stolos Authors.
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
)

type ReservedOrInvalidNamespaceTestcase struct {
	Name  string
	Error bool
}

func TestIsReservedOrInvalidNamespace(t *testing.T) {
	for _, testcase := range []ReservedOrInvalidNamespaceTestcase{
		{"foo-bar", false},
		{"Foo-Bar", true},
		{"Foo_Bar", true},
		{"-Foo_Bar", true},
		{"Foo_Bar-", true},
		{"ALL-CAPS", true},
		{"-foo-bar", true},
		{"foo-bar-", true},
		{"kube-foo", true},
		{"default", true},
		{"stolos-system", true},
	} {
		error := IsReservedOrInvalidNamespace(testcase.Name)
		if error == nil && testcase.Error {
			t.Errorf("Expected error but didn't get one, testing %q", testcase.Name)
		}

		if error != nil && !testcase.Error {
			t.Errorf("Unexpected testing %q", testcase.Name)
		}
	}
}
