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
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions/resource"
)

type validator struct {
	seen map[schema.GroupVersionKind]bool
	err  error
}

// newValidator creates a validator for Nomos-specific constraints on Kubernetes objects.
// err holds the first encountered error. Subsequent errors are short-circuited.
// Error messages here must be as specific and actionable as possible.
func newValidator() *validator {
	return &validator{seen: make(map[schema.GroupVersionKind]bool)}
}

// HasNamespace validates that the object in info has the given namespace.
func (v *validator) HasNamespace(info *resource.Info, namespace string) *validator {
	if v.err != nil {
		return v
	}
	if info.Namespace != namespace {
		v.err = errors.Errorf("metadata.namespace must be set to %q instead of %q in %s", namespace, info.Namespace, info.Source)
	}
	return v
}

// MarkSeen marks the given type as seen.
func (v *validator) MarkSeen(t schema.GroupVersionKind) *validator {
	v.seen[t] = true
	return v
}

// HaveSeen validates that the MarkSeen method is called at least once with the given type.
func (v *validator) HaveSeen(t schema.GroupVersionKind) *validator {
	if v.err != nil {
		return v
	}
	if !v.seen[t] {
		v.err = errors.Errorf("must have a object with type %#v", t)
	}
	return v
}

// ObjectedDisallowedContext indicates that an object of given type is not allowed in this directory.
func (v *validator) ObjectDisallowedInContext(info *resource.Info, t schema.GroupVersionKind) *validator {
	if v.err != nil {
		return v
	}
	v.err = errors.Errorf("object of type %#v is not allowed in %s", t, info.Source)
	return v
}
