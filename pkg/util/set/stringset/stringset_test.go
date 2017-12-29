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

package stringset

import "testing"

func TestBasicOps(t *testing.T) {
	set := New()

	values := []string{"foo", "bar", "baz"}
	for _, value := range values {
		set.Add(value)
		if !set.Contains(value) {
			t.Errorf("Set should contain %s", value)
		}
	}

	if len(set.ToSlice()) != 3 {
		t.Errorf("Slice should be of size 3, got: %#v", set)
	}
	if set.Size() != 3 {
		t.Errorf("Size should be 3, got: %#v", set)
	}

	set2 := New()
	set2.AddSlice(set.ToSlice())

	if !set.Equals(set2) {
		t.Errorf("Sets should be equal: %#v != %#v", set, set2)
	}

	set3 := New()
	set.ForEach(func(value string) {
		set3.Add(value)
	})

	if !set.Equals(set3) {
		t.Errorf("Sets should be equal: %#v != %#v", set, set3)
	}

	for _, value := range values {
		set.Remove(value)
		if set.Contains(value) {
			t.Errorf("Set should not contain %s", value)
		}

		if set.Equals(set2) {
			t.Errorf("Sets should not be equal: %#v == %#v", set, set2)
		}
	}

	if !New().Equals(New()) {
		t.Errorf("Two empty sets should be equivalent")
	}
}

type SetOperationTestcase struct {
	LeftHandSide       []string
	RightHandSide      []string
	DifferenceResult   []string
	IntersectionResult []string
}

func TestSetOperations(t *testing.T) {
	for idx, testcase := range []SetOperationTestcase{
		{
			LeftHandSide:       []string{},
			RightHandSide:      []string{},
			DifferenceResult:   []string{},
			IntersectionResult: []string{},
		},
		{
			LeftHandSide:       []string{},
			RightHandSide:      []string{"foo", "bar"},
			DifferenceResult:   []string{},
			IntersectionResult: []string{},
		},
		{
			LeftHandSide:       []string{"foo", "bar"},
			RightHandSide:      []string{},
			DifferenceResult:   []string{"foo", "bar"},
			IntersectionResult: []string{},
		},
		{
			LeftHandSide:       []string{"foo", "bar"},
			RightHandSide:      []string{"bar"},
			DifferenceResult:   []string{"foo"},
			IntersectionResult: []string{"bar"},
		},
		{
			LeftHandSide:       []string{"foo", "bar"},
			RightHandSide:      []string{"foo", "bar"},
			DifferenceResult:   []string{},
			IntersectionResult: []string{"foo", "bar"},
		},
		{
			LeftHandSide:       []string{"foo", "bar"},
			RightHandSide:      []string{"baz", "foobar"},
			DifferenceResult:   []string{"foo", "bar"},
			IntersectionResult: []string{},
		},
		{
			LeftHandSide:       []string{"foo", "bar"},
			RightHandSide:      []string{"foo", "bar"},
			DifferenceResult:   []string{},
			IntersectionResult: []string{"foo", "bar"},
		},
	} {
		lhs := NewFromSlice(testcase.LeftHandSide)
		rhs := NewFromSlice(testcase.RightHandSide)

		expectedDifferenceResult := NewFromSlice(testcase.DifferenceResult)
		differenceResult := lhs.Difference(rhs)
		if !differenceResult.Equals(expectedDifferenceResult) {
			t.Errorf("Unexpected difference result %#v in testcase %d %#v", differenceResult, idx, testcase)
		}

		expectedIntersectionResult := NewFromSlice(testcase.IntersectionResult)
		intersectionResult := lhs.Intersection(rhs)
		if !intersectionResult.Equals(expectedIntersectionResult) {
			t.Errorf("Unexpected intersection result %#v in testcase %d %#v", intersectionResult, idx, testcase)
		}
	}
}
