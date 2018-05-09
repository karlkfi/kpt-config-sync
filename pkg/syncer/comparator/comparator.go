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

	"github.com/google/nomos/pkg/client/action"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Type indicates the state of the given resource
type Type string

const (
	// Add indicates the resource is declared, but missing on the API server
	Add = Type("add")
	// Update indicates the resource is declared, but different on the API server
	Update = Type("update")
	// Delete indicates the resource is not declared, but exists on the API server
	Delete = Type("delete")
)

// Equals is a function that takes two objects then compares them while ignoring the object meta
// labels and annotations.
type Equals func(meta_v1.Object, meta_v1.Object) bool

// Diff is resource where Declared and Actual do not match.
type Diff struct {
	Name     string
	Type     Type
	Declared meta_v1.Object
	Actual   meta_v1.Object
}

// Comparator handles comparing declared to actual on arbitrary resources given a type-specific
// equlivalence function.
type Comparator struct {
	equals Equals
}

// New returns a new reconciler which will use the given equals function
func New(equals Equals) *Comparator {
	return &Comparator{equals: equals}
}

// Compare returns the diffs between declared and actual state. The diffs will be returned in an
// arbitrary order.
func (s *Comparator) Compare(declared []meta_v1.Object, actual []meta_v1.Object) []*Diff {
	return Compare(s.equals, declared, actual)
}

// Compare returns the diffs between declared and actual state. The diffs will be returned in an
// arbitrary order.
func Compare(equals Equals, declared []meta_v1.Object, existing []meta_v1.Object) []*Diff {
	var diffs []*Diff

	decls := map[string]meta_v1.Object{}
	for _, decl := range declared {
		name := decl.GetName()
		if _, found := decls[name]; found {
			panic(invalidInput{desc: fmt.Sprintf("Got duplicate decl for %q", name)})
		}
		decls[name] = decl
	}

	actuals := map[string]meta_v1.Object{}
	for _, item := range existing {
		actuals[item.GetName()] = item
	}

	for name, decl := range decls {
		if actual, found := actuals[name]; !found {
			// in decl, not in actual
			diffs = append(diffs, &Diff{
				Name:     name,
				Type:     Add,
				Declared: decl,
				Actual:   nil,
			})
		} else {
			// We check equals on content body, but meta subset on the labels / annotations because
			// it's possible for another controller to add some form of annotation or label to the object
			// without changing the intent of what's going on. For an example of this, see ClusterRole
			// aggregation.
			if !equals(decl, actual) || !action.MetaSubset(actual, decl) {
				diffs = append(diffs, &Diff{
					Name:     name,
					Type:     Update,
					Declared: decl,
					Actual:   actual,
				})
			}
		}
	}

	for name, actual := range actuals {
		if _, found := decls[name]; !found {
			// Not in declared, in actuals
			diffs = append(diffs, &Diff{
				Name:     name,
				Type:     Delete,
				Declared: nil,
				Actual:   actual,
			})
		}
	}
	return diffs
}

type invalidInput struct {
	desc string
}

func (i *invalidInput) String() string {
	return i.desc
}
