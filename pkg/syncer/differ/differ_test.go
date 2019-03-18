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

package differ

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	nomosv1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var converter = runtime.DefaultUnstructuredConverter

type TestItem struct {
	name        string
	value       string
	annotations map[string]string
}

func (ti TestItem) Object(t *testing.T) *unstructured.Unstructured {
	obj := &nomosv1.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ti.name,
			Annotations: ti.annotations,
		},
		Spec: nomosv1.ClusterConfigSpec{
			ImportToken: ti.value,
		},
	}
	u, err := converter.ToUnstructured(obj)
	if err != nil {
		t.Fatal(err)
	}
	return &unstructured.Unstructured{Object: u}
}

type TestItems []TestItem

func (ti TestItems) Objects(t *testing.T) []*unstructured.Unstructured {
	ret := make([]*unstructured.Unstructured, len(ti))
	for idx, item := range ti {
		ret[idx] = item.Object(t)
	}
	return ret
}

type TestCase struct {
	name              string
	decls             TestItems
	otherVersionDecls TestItems
	actuals           TestItems
	expect            []*Diff
	expectPanic       bool
}

func TestComparator(t *testing.T) {
	testcases := []TestCase{
		{
			name: "Empty sets",
		},
		{
			name: "Only declared",
			decls: TestItems{
				TestItem{name: "foo", value: "1"},
			},
			expect: []*Diff{
				{
					Name:     "foo",
					Type:     Add,
					Declared: TestItem{name: "foo", value: "1"}.Object(t),
					Actual:   nil,
				},
			},
		},
		{
			name: "Only actual",
			actuals: TestItems{
				TestItem{name: "foo", value: "2"},
			},
			expect: []*Diff{
				{
					Name:     "foo",
					Type:     Delete,
					Declared: nil,
					Actual:   TestItem{name: "foo", value: "2"}.Object(t),
				},
			},
		},
		{
			name: "Don't delete resources declared in other versions",
			otherVersionDecls: TestItems{
				TestItem{name: "foo", value: "2"},
				TestItem{name: "bar", value: "4"},
			},
			actuals: TestItems{
				TestItem{name: "foo", value: "2"},
				TestItem{name: "bar", value: "2"},
			},
		},
		{
			name: "Differing element value",
			decls: TestItems{
				TestItem{name: "foo", value: "1"},
			},
			actuals: TestItems{
				TestItem{name: "foo", value: "2"},
			},
			expect: []*Diff{
				{
					Name:     "foo",
					Type:     Update,
					Declared: TestItem{name: "foo", value: "1"}.Object(t),
					Actual:   TestItem{name: "foo", value: "2"}.Object(t),
				},
			},
		},
		{
			name: "Mixture",
			decls: TestItems{
				TestItem{name: "foo", value: "1"},
				TestItem{name: "bar", value: "2"},
				TestItem{name: "baz", value: "3"},
			},
			actuals: TestItems{
				TestItem{name: "foo", value: "2"},
				TestItem{name: "bar", value: "2"},
				TestItem{name: "buffalo", value: "4"},
			},
			expect: []*Diff{
				{
					Name:     "foo",
					Type:     Update,
					Declared: TestItem{name: "foo", value: "1"}.Object(t),
					Actual:   TestItem{name: "foo", value: "2"}.Object(t),
				},
				{
					Name:     "bar",
					Type:     Update,
					Declared: TestItem{name: "bar", value: "2"}.Object(t),
					Actual:   TestItem{name: "bar", value: "2"}.Object(t),
				},
				{
					Name:     "baz",
					Type:     Add,
					Declared: TestItem{name: "baz", value: "3"}.Object(t),
					Actual:   nil,
				},
				{
					Name:     "buffalo",
					Type:     Delete,
					Declared: nil,
					Actual:   TestItem{name: "buffalo", value: "4"}.Object(t),
				},
			},
		},
		{
			name: "panic on duplicate decl names",
			decls: TestItems{
				TestItem{name: "pod-creator", value: "2"},
				TestItem{name: "pod-creator", value: "2"},
			},
			actuals: TestItems{
				TestItem{name: "pod-creator", value: "2"},
			},
			expectPanic: true,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			defer func() {
				if x := recover(); x != nil {
					if _, ok := x.(invalidInput); ok && testcase.expectPanic {
						return
					}
					panic(x)
				}
			}()

			declared := testcase.decls.Objects(t)
			actuals := testcase.actuals.Objects(t)
			allVersionsDeclared := map[string]bool{}
			for _, r := range append(declared, testcase.otherVersionDecls.Objects(t)...) {
				allVersionsDeclared[r.GetName()] = true
			}

			diff := Diffs(declared, allVersionsDeclared, actuals)

			for _, expect := range testcase.expect {
				var found bool
				for _, actual := range diff {
					if reflect.DeepEqual(expect, actual) {
						found = true
					}
				}
				if !found {
					t.Errorf("expected diff %#v missing from actual", spew.Sdump(expect))
					fmt.Printf("expect:\n%s\ndiff:\n%s\n", spew.Sdump(expect), spew.Sdump(diff))
				}
			}
			if len(diff) != len(testcase.expect) {
				t.Errorf("Expected was different length than actual")
			}
		})
	}
}

func TestActualResourceIsManaged(t *testing.T) {
	testcases := []struct {
		name        string
		actualNil   bool
		annotations map[string]string
		want        ManagementState
	}{
		{
			name:      "nil actual",
			actualNil: true,
			want:      Unmanaged,
		},
		{
			name: "nil annotations",
			want: Unset,
		},
		{
			name:        "invalid value",
			annotations: map[string]string{nomosv1.ResourceManagementKey: "invalid"},
			want:        Invalid,
		},
		{
			name:        "disabled value",
			annotations: map[string]string{nomosv1.ResourceManagementKey: nomosv1.ResourceManagementDisabled},
			want:        Unmanaged,
		},
		{
			name:        "enabled value",
			annotations: map[string]string{nomosv1.ResourceManagementKey: nomosv1.ResourceManagementEnabled},
			want:        Managed,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			d := Diff{
				Actual: TestItem{annotations: testcase.annotations}.Object(t),
			}
			if testcase.actualNil {
				d.Actual = nil
			}
			got := d.ManagementState()
			if got != testcase.want {
				t.Errorf("want actual is %s, got %s", testcase.want, got)
			}
		})
	}
}
