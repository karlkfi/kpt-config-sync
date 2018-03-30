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

package reconciler

import (
	"github.com/golang/glog"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Type indicates the state of the given resource
type Type string

const (
	// Removed indicates the resource is declared, but missing on the API server
	Removed = Type("removed")
	// Modified indicates the resource is declared, but different on the API server
	Modified = Type("modified")
	// Added indicates the resource is not declared, but exists on the API server
	Added = Type("added")
)

// Equals is a function that takes two objects then compares them while ignoring the object meta
// labels and annotations.
type Equals func(meta_v1.Object, meta_v1.Object) bool

// Diff is resource where Declared and Actual do not match.
type Diff struct {
	Type     Type
	Declared meta_v1.Object
	Actual   meta_v1.Object
}

// Reconciler handles comparing declared to actual on arbitrary resources given a type-specific
// equlivalence function.
type Reconciler struct {
	equals Equals
}

// New returns a new reconciler which will use the given equals function
func New(equals Equals) *Reconciler {
	return &Reconciler{equals: equals}
}

// Compare returns the diffs between declared and actual state. The diffs will be returned in an
// arbitrary order.
func (s *Reconciler) Compare(declared []meta_v1.Object, actual []meta_v1.Object) []*Diff {
	glog.Fatal("not implemented")
	return nil
}
