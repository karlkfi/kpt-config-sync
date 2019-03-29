/*
Copyright 2018 The CSP Config Management Authors.

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

package differ

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func buildUnstructured(opts ...func(*unstructured.Unstructured)) *unstructured.Unstructured {
	result := &unstructured.Unstructured{}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

func name(s string) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		u.SetName(s)
	}
}

func managed(s string) func(*unstructured.Unstructured) {
	return func(u *unstructured.Unstructured) {
		annotations := u.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[v1.ResourceManagementKey] = s
		u.SetAnnotations(annotations)
	}
}

func TestComparator(t *testing.T) {
	testcases := []struct {
		name                string
		declared            []*unstructured.Unstructured
		actuals             []*unstructured.Unstructured
		allDeclaredVersions map[string]bool
		expect              map[string]*Diff
		expectTypes         map[string]Type
		expectPanic         bool
	}{
		{
			name:        "empty returns empty",
			expect:      map[string]*Diff{},
			expectTypes: map[string]Type{},
		},
		{
			name: "declared and not in actual returns create",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
			},
			allDeclaredVersions: map[string]bool{"foo": true},
			expect: map[string]*Diff{
				"foo": {
					Name:     "foo",
					Declared: buildUnstructured(name("foo")),
				},
			},
			expectTypes: map[string]Type{
				"foo": Create,
			},
		},
		{
			name: "declared and in actual and management enabled returns update",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
			},
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed(v1.ResourceManagementEnabled)),
			},
			allDeclaredVersions: map[string]bool{"foo": true},
			expect: map[string]*Diff{
				"foo": {
					Name:     "foo",
					Declared: buildUnstructured(name("foo")),
					Actual:   buildUnstructured(name("foo"), managed(v1.ResourceManagementEnabled)),
				},
			},
			expectTypes: map[string]Type{
				"foo": Update,
			},
		},
		{
			name: "declared and in actual and management diabled returns noop",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
			},
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed(v1.ResourceManagementDisabled)),
			},
			allDeclaredVersions: map[string]bool{"foo": true},
			expect: map[string]*Diff{
				"foo": {
					Name:     "foo",
					Declared: buildUnstructured(name("foo")),
					Actual:   buildUnstructured(name("foo"), managed(v1.ResourceManagementDisabled)),
				},
			},
			expectTypes: map[string]Type{
				"foo": NoOp,
			},
		},
		{
			name: "declared and in actual and management invalid returns error",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
			},
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed("error")),
			},
			allDeclaredVersions: map[string]bool{"foo": true},
			expect: map[string]*Diff{
				"foo": {
					Name:     "foo",
					Declared: buildUnstructured(name("foo")),
					Actual:   buildUnstructured(name("foo"), managed("error")),
				},
			},
			expectTypes: map[string]Type{
				"foo": Error,
			},
		},
		{
			name: "not declared and in actual and management enabled returns delete",
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed(v1.ResourceManagementEnabled)),
			},
			expect: map[string]*Diff{
				"foo": {
					Name:   "foo",
					Actual: buildUnstructured(name("foo"), managed(v1.ResourceManagementEnabled)),
				},
			},
			expectTypes: map[string]Type{
				"foo": Delete,
			},
		},
		{
			name: "not declared and in actual and management enabled, but in different version returns no diff",
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed(v1.ResourceManagementEnabled)),
			},
			allDeclaredVersions: map[string]bool{"foo": true},
			expect:              map[string]*Diff{},
			expectTypes:         map[string]Type{},
		},
		{
			name: "not declared and in actual and management disabled returns noop",
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed(v1.ResourceManagementDisabled)),
			},
			expect: map[string]*Diff{
				"foo": {
					Name:   "foo",
					Actual: buildUnstructured(name("foo"), managed(v1.ResourceManagementDisabled)),
				},
			},
			expectTypes: map[string]Type{
				"foo": NoOp,
			},
		},
		{
			name: "not declared and in actual and management invalid returns error",
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("foo"), managed("error")),
			},
			expect: map[string]*Diff{
				"foo": {
					Name:   "foo",
					Actual: buildUnstructured(name("foo"), managed("error")),
				},
			},
			expectTypes: map[string]Type{
				"foo": Error,
			},
		},
		{
			name: "multiple diff types works",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
				buildUnstructured(name("bar")),
				buildUnstructured(name("qux")),
			},
			actuals: []*unstructured.Unstructured{
				buildUnstructured(name("bar"), managed(v1.ResourceManagementEnabled)),
				buildUnstructured(name("qux"), managed(v1.ResourceManagementDisabled)),
				buildUnstructured(name("mun"), managed(v1.ResourceManagementEnabled)),
			},
			allDeclaredVersions: map[string]bool{
				"foo": true, "bar": true, "qux": true,
			},
			expect: map[string]*Diff{
				"foo": {
					Name:     "foo",
					Declared: buildUnstructured(name("foo")),
				},
				"bar": {
					Name:     "bar",
					Declared: buildUnstructured(name("bar")),
					Actual:   buildUnstructured(name("bar"), managed(v1.ResourceManagementEnabled)),
				},
				"qux": {
					Name:     "qux",
					Declared: buildUnstructured(name("qux")),
					Actual:   buildUnstructured(name("qux"), managed(v1.ResourceManagementDisabled)),
				},
				"mun": {
					Name:   "mun",
					Actual: buildUnstructured(name("mun"), managed(v1.ResourceManagementEnabled)),
				},
			},
			expectTypes: map[string]Type{
				"foo": Create,
				"bar": Update,
				"qux": NoOp,
				"mun": Delete,
			},
		},
		{
			name: "duplicate declarations panics",
			declared: []*unstructured.Unstructured{
				buildUnstructured(name("foo")),
				buildUnstructured(name("foo")),
			},
			expectPanic: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if x := recover(); x != nil {
					if _, ok := x.(invalidInput); ok && tc.expectPanic {
						return
					}
					t.Fatal(x)
				}
			}()

			diffs := Diffs(tc.declared, tc.actuals, tc.allDeclaredVersions)

			if len(tc.declared) > 0 {
				fmt.Printf("%v\n", tc.declared[0].Object)
				fmt.Println("name: ", tc.declared[0].GetName())
			}

			diffsMap := make(map[string]*Diff)
			diffTypesMap := make(map[string]Type)
			for _, diff := range diffs {
				fmt.Println(diff)
				diffsMap[diff.Name] = diff
				diffTypesMap[diff.Name] = diff.Type()
			}

			if tDiff := cmp.Diff(tc.expect, diffsMap); tDiff != "" {
				t.Fatal(tDiff)
			}

			if tDiff := cmp.Diff(tc.expectTypes, diffTypesMap); tDiff != "" {
				t.Fatal(tDiff)
			}
		})
	}
}
