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

// Package differ contains code for diffing sync-enabled resources, not
// necessarily known at compile time.
package differ

import (
	"fmt"

	"github.com/google/nomos/pkg/syncer/labeling"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
type Equals func(*unstructured.Unstructured, *unstructured.Unstructured) bool

// Diff is resource where Declared and Actual do not match.
type Diff struct {
	Name     string
	Type     Type
	Declared *unstructured.Unstructured
	Actual   *unstructured.Unstructured
}

// ActualResourceIsManaged returns true if the Actual resource in the Diff has a management label.
func (d Diff) ActualResourceIsManaged() bool {
	if d.Actual == nil {
		return false
	}

	labels := d.Actual.GetLabels()
	if labels == nil {
		return false
	}

	value, ok := labels[labeling.ResourceManagementKey]
	if !ok {
		return false
	}

	// TODO(120490008): return value for indicating invalid label value.
	return value == labeling.Enabled
}

// Diffs returns the diffs between declared and actual state. We generate a diff for each GroupVersionKind.
// The actual resources are for all versions of a GroupKind and the declared resources are for a particular GroupKind.
// We need to ensure there is not a declared resource across all possible versions before we delete it.
// The diffs will be returned in an arbitrary order.
func Diffs(declared []*unstructured.Unstructured, allDeclaredVersions map[string]bool,
	existing []*unstructured.Unstructured) []*Diff {

	decls := asMap(declared)
	actuals := map[string]*unstructured.Unstructured{}
	for _, item := range existing {
		actuals[item.GetName()] = item
	}

	var diffs []*Diff
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
			// Applier handles comparison between declared and actual: always update.
			diffs = append(diffs, &Diff{
				Name:     name,
				Type:     Update,
				Declared: decl,
				Actual:   actual,
			})
		}
	}

	for name, actual := range actuals {
		if !allDeclaredVersions[name] {
			// Not in any declared version, in actuals
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

func asMap(us []*unstructured.Unstructured) map[string]*unstructured.Unstructured {
	m := map[string]*unstructured.Unstructured{}
	for _, u := range us {
		name := u.GetName()
		if _, found := m[name]; found {
			panic(invalidInput{desc: fmt.Sprintf("Got duplicate decl for %q", name)})
		}
		m[name] = u
	}
	return m
}

type invalidInput struct {
	desc string
}

func (i *invalidInput) String() string {
	return i.desc
}
