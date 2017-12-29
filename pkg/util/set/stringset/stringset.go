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

// Package stringset provides an implementation of a set of strings since this doesn't seem to exist
// and I wanted to avoid existing libraries that rely on interface{}
package stringset

// StringSet implements a set of strings.
type StringSet struct {
	set map[string]bool
}

// New Creates a new StringSet
func New() *StringSet {
	return &StringSet{
		set: map[string]bool{},
	}
}

// NewFromSlice creates a new set from a slice of strings.
func NewFromSlice(values []string) *StringSet {
	ret := New()
	ret.AddSlice(values)
	return ret
}

// Add adds an item to the set
func (s *StringSet) Add(value string) {
	s.set[value] = true
}

// AddSlice adds all the values in a slice
func (s *StringSet) AddSlice(values []string) {
	for _, value := range values {
		s.Add(value)
	}
}

// Remove removes an item from the set
func (s *StringSet) Remove(value string) {
	delete(s.set, value)
}

// Contains returns true if the set contains the given value
func (s *StringSet) Contains(value string) bool {
	return s.set[value]
}

// Equals returns true if the other set is equivalent to this set.
func (s *StringSet) Equals(other *StringSet) bool {
	if len(s.set) != len(other.set) {
		return false
	}

	for value := range s.set {
		if !other.set[value] {
			return false
		}
	}

	return true
}

// Size returns the number of elements in the set
func (s *StringSet) Size() int {
	return len(s.set)
}

// ForEach calls cb on each element in the set.
func (s *StringSet) ForEach(cb func(string)) {
	for value := range s.set {
		cb(value)
	}
}

// ToSlice returns the set as a slice of strings.
func (s *StringSet) ToSlice() []string {
	idx := 0
	ret := make([]string, len(s.set))
	for value := range s.set {
		ret[idx] = value
		idx++
	}
	return ret
}

// Difference computes the set difference of (s - other).
func (s *StringSet) Difference(other *StringSet) *StringSet {
	differenceSet := New()
	for value := range s.set {
		if !other.set[value] {
			differenceSet.Add(value)
		}
	}
	return differenceSet
}

// Intersection computes the set intersection between s and other
func (s *StringSet) Intersection(other *StringSet) *StringSet {
	intersection := New()
	for value := range s.set {
		if other.set[value] {
			intersection.Add(value)
		}
	}
	return intersection
}
