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

package comparator

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TestObject struct {
	meta_v1.ObjectMeta
	meta_v1.TypeMeta

	value string
}

func testEquals(lhsObj meta_v1.Object, rhsObj meta_v1.Object) bool {
	lhs := lhsObj.(*TestObject)
	rhs := rhsObj.(*TestObject)
	return lhs.value == rhs.value
}

type TestItem struct {
	name        string
	value       string
	labels      map[string]string
	annotations map[string]string
}

func (t TestItem) Object() meta_v1.Object {
	return &TestObject{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:        t.name,
			Labels:      t.labels,
			Annotations: t.annotations,
		},
		value: t.value,
	}
}

type TestItems []TestItem

func (t TestItems) Objects() []meta_v1.Object {
	ret := make([]meta_v1.Object, len(t))
	for idx, item := range t {
		ret[idx] = item.Object()
	}
	return ret
}

type TestCase struct {
	name        string
	decls       TestItems
	actuals     TestItems
	expect      []*Diff
	expectPanic bool
}

func TestComparitor(t *testing.T) {
	testcases := []TestCase{
		TestCase{
			name: "Empty sets",
		},
		TestCase{
			name: "Only declared",
			decls: TestItems{
				TestItem{name: "foo", value: "1"},
			},
			expect: []*Diff{
				&Diff{
					Name:     "foo",
					Type:     Add,
					Declared: TestItem{name: "foo", value: "1"}.Object(),
					Actual:   nil,
				},
			},
		},
		TestCase{
			name: "Only actual",
			actuals: TestItems{
				TestItem{name: "foo", value: "2"},
			},
			expect: []*Diff{
				&Diff{
					Name:     "foo",
					Type:     Delete,
					Declared: nil,
					Actual:   TestItem{name: "foo", value: "2"}.Object(),
				},
			},
		},
		TestCase{
			name: "Differing element value",
			decls: TestItems{
				TestItem{name: "foo", value: "1"},
			},
			actuals: TestItems{
				TestItem{name: "foo", value: "2"},
			},
			expect: []*Diff{
				&Diff{
					Name:     "foo",
					Type:     Update,
					Declared: TestItem{name: "foo", value: "1"}.Object(),
					Actual:   TestItem{name: "foo", value: "2"}.Object(),
				},
			},
		},
		TestCase{
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
				&Diff{
					Name:     "foo",
					Type:     Update,
					Declared: TestItem{name: "foo", value: "1"}.Object(),
					Actual:   TestItem{name: "foo", value: "2"}.Object(),
				},
				&Diff{
					Name:     "baz",
					Type:     Add,
					Declared: TestItem{name: "baz", value: "3"}.Object(),
					Actual:   nil,
				},
				&Diff{
					Name:     "buffalo",
					Type:     Delete,
					Declared: nil,
					Actual:   TestItem{name: "buffalo", value: "4"}.Object(),
				},
			},
		},
		TestCase{
			name: "b/79438010 - error on duplicate decl names",
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

			declared := testcase.decls.Objects()
			actuals := testcase.actuals.Objects()

			cmp := New(testEquals)
			diff := cmp.Compare(declared, actuals)

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
