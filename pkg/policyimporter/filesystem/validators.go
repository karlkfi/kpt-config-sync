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

package filesystem

import (
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

type validator struct {
	seen map[meta_v1.TypeMeta]bool
	err  error
}

// newValidator creates a validator for Stolos-specific constraints on Kubernetes objects.
// err holds the first encountered error. Subsequent errors are short-circuited.
// Error messages here must be as specific and actionable as possible.
func newValidator() *validator {
	return &validator{seen: make(map[meta_v1.TypeMeta]bool)}
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

// HasName validates that the object has the given name.
func (v *validator) HasName(info *resource.Info, name string) *validator {
	if v.err != nil {
		return v
	}
	if info.Name != name {
		v.err = errors.Errorf("metadata.name must be set to %q instead of %q in %s", name, info.Name, info.Source)
	}
	return v
}

// MarkSeen marks the given type as seen.
func (v *validator) MarkSeen(t meta_v1.TypeMeta) *validator {
	v.seen[t] = true
	return v
}

// HaveNotSeen validates that the MarkSeen method is called at most once with the given type.
func (v *validator) HaveNotSeen(t meta_v1.TypeMeta) *validator {
	if v.err != nil {
		return v
	}
	if v.seen[t] {
		v.err = errors.Errorf("cannot have multiple objects with type %#v", t)
	}
	return v
}

// HaveSeen validates that the MarkSeen method is called at least once with the given type.
func (v *validator) HaveSeen(t meta_v1.TypeMeta) *validator {
	if v.err != nil {
		return v
	}
	if !v.seen[t] {
		v.err = errors.Errorf("must have a object with type %#v", t)
	}
	return v
}

// ObjectedDisallowedContext indicates that an object of given type is not allowed in this directory.
func (v *validator) ObjectDisallowedInContext(info *resource.Info, t meta_v1.TypeMeta) *validator {
	if v.err != nil {
		return v
	}
	v.err = errors.Errorf("object of type %#v is not allowed in %s", t, info.Source)
	return v
}
