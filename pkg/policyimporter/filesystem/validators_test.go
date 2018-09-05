// Reviewed by sunilarora
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

package filesystem

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

type validatorsTestCase struct {
	testName      string
	v             *validator
	expectedError bool
}

var namespaceType = schema.GroupVersionKind{Version: "v1", Kind: "Namespace"}

var testCases = []validatorsTestCase{
	{"HasName valid", newValidator().HasName(&resource.Info{Name: "foo"}, "foo"), false},
	{"HasName invalid", newValidator().HasName(&resource.Info{Name: "foo"}, "bar"), true},
	{"HasNamespace valid", newValidator().HasNamespace(&resource.Info{Namespace: "foo"}, "foo"), false},
	{"HasNamespace invalid", newValidator().HasNamespace(&resource.Info{Namespace: "foo"}, "bar"), true},
	{"Keep first error", newValidator().HasNamespace(&resource.Info{Namespace: "foo"}, "bar").HasName(&resource.Info{Name: "foo"}, "foo"), true},
	{"HaveSeen valid", newValidator().MarkSeen(namespaceType).HaveSeen(namespaceType), false},
	{"HaveSeen invalid", newValidator().HaveSeen(namespaceType), true},
	{"HaveNotSeen valid", newValidator().HaveNotSeen(namespaceType), false},
	{"HaveNotSeen invalid", newValidator().MarkSeen(namespaceType).HaveNotSeen(namespaceType), true},
	{"ObjectDisallowedInContext", newValidator().ObjectDisallowedInContext(&resource.Info{Source: "some/source"}, namespaceType), true},
}

func TestValidator(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			if !tc.expectedError && tc.v.err != nil {
				t.Fatalf("Expected error: %v", tc.v.err)
			}
			if tc.expectedError && tc.v.err == nil {
				t.Fatalf("Unexpected error")
			}
		})
	}
}
