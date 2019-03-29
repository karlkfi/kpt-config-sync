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

// Package differ contains code for diffing sync-enabled resources, not
// necessarily known at compile time.
package differ

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Type indicates the state of the given resource
type Type string

const (
	// NoOp indicates that no action should be taken.
	NoOp = Type("no-op")

	// Create indicates the resource should be created.
	Create = Type("create")

	// Update indicates the resource is declared and is on the API server, so we should
	// calculate a patch and apply it.
	Update = Type("update")

	// Delete indicates the resource should be deleted.
	Delete = Type("delete")

	// Error indicates the resource's management label in the API server is invalid.
	Error = Type("error")
)

// Equals is a function that takes two objects then compares them while ignoring the object meta
// labels and annotations.
type Equals func(*unstructured.Unstructured, *unstructured.Unstructured) bool

// Diff is resource where Declared and Actual do not match.
type Diff struct {
	// Name is the name of the resource this diff is for.
	Name string
	// Declared is the resource as it exists in the repository.
	Declared *unstructured.Unstructured
	// Actual is the resource as it exists in the cluster.
	Actual *unstructured.Unstructured
}

// Type returns the type of the difference between the repository and the API Server.
func (d Diff) Type() Type {
	if d.Declared != nil {
		// The resource is in the repository.
		if d.Actual == nil {
			// The resource is NOT in the API Server.
			// Therefore, create the resource.
			return Create
		}

		// The resource is in the API Server.
		switch d.Actual.GetAnnotations()[v1.ResourceManagementKey] {
		case v1.ResourceManagementEnabled, "":
			// Update resource with management explicitly enabled or unset.
			return Update
		case v1.ResourceManagementDisabled:
			return NoOp
		}
		// The annotation in the cluster is invalid, so show an error.
		return Error
	}
	// The resource is NOT in the repository.

	if d.Actual != nil {
		// The resource is in the API Server.
		switch d.Actual.GetAnnotations()[v1.ResourceManagementKey] {
		case v1.ResourceManagementEnabled:
			// Delete resource with management enabled on API Server.
			return Delete
		case v1.ResourceManagementDisabled, "":
			// Do not delete resource with management explicitly disabled or unset.
			return NoOp
		}
		// The annotation in the cluster is invalid, so show an error.
		return Error
	}

	// The resource is neither on the API Server nor in the repo, so do nothing.
	return NoOp
}

// Diffs returns the diffs between declared and actual state. We generate a diff for each GroupVersionKind.
// The actual resources are for all versions of a GroupKind and the declared resources are for a particular GroupKind.
// We need to ensure there is not a declared resource across all possible versions before we delete it.
// The diffs will be returned in an arbitrary order.
func Diffs(declared []*unstructured.Unstructured, actuals []*unstructured.Unstructured, allDeclaredVersions map[string]bool) []*Diff {
	actualsMap := map[string]*unstructured.Unstructured{}
	for _, obj := range actuals {
		// Assume no collisions among resources on API Server.
		actualsMap[obj.GetName()] = obj
	}

	decls := asMap(declared)
	var diffs []*Diff
	for name, decl := range decls {
		diffs = append(diffs, &Diff{
			Name:     name,
			Actual:   actualsMap[name],
			Declared: decl,
		})
	}
	for name, actual := range actualsMap {
		if !allDeclaredVersions[name] {
			// Not in any declared version, but on the API Server.
			diffs = append(diffs, &Diff{
				Name:   name,
				Actual: actual,
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
