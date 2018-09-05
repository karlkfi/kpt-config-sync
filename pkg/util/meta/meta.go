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

// Package meta has utilities for validating the ObjectMeta and TypeMeta structs that exist
// on kubernetes api objects.
package meta

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/validation/path"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/core/validation"
)

// apiObject is a kubernetes API object type composed of meta/v1/Object and a subset of runtime.Object.
type apiObject interface {
	metav1.Object
	GetObjectKind() schema.ObjectKind
}

// Validator checks the ObjectMeta (metadata) field of a kubernetes API object for sanity.
type Validator struct {
	// Flattened indicates that this resource will be flattened.
	Flattened bool
}

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{}
}

func makeValidator(f func(string, bool) []string) func(string) []string {
	return func(n string) []string {
		return f(n, false)
	}
}

func (v *Validator) getEffectiveName(namespace string, obj apiObject) string {
	if v.Flattened {
		return fmt.Sprintf("%s.%s", namespace, obj.GetName())
	}
	return obj.GetName()
}

func (v *Validator) validateName(namespace string, obj apiObject) error {
	name := v.getEffectiveName(namespace, obj)

	var validator func(string) []string
	switch obj.(type) {
	case *rbacv1.Role, *rbacv1.RoleBinding, *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding:
		validator = path.IsValidPathSegmentName

	default:
		validator = makeValidator(validation.NameIsDNSSubdomain)
	}

	if msgs := validator(name); msgs != nil {
		return errors.Errorf("invalid name on %q: %s", name, strings.Join(msgs, ", "))
	}
	return nil
}

func toAPIObjectList(resourceList interface{}) []apiObject {
	rlv := reflect.ValueOf(resourceList)
	if rlv.Type().Kind() != reflect.Slice {
		panic(fmt.Sprintf("invalid type, expected slice, got %v: %v", rlv.Type(), resourceList))
	}

	if rlv.IsNil() || rlv.Len() == 0 {
		return nil
	}

	var getFunc func(val reflect.Value) apiObject
	switch rlv.Index(0).Type().Kind() {
	case reflect.Ptr:
		getFunc = func(val reflect.Value) apiObject { return val.Interface().(apiObject) }
	case reflect.Struct:
		getFunc = func(val reflect.Value) apiObject { return val.Addr().Interface().(apiObject) }
	}

	apiObjects := make([]apiObject, rlv.Len())
	for i := 0; i < rlv.Len(); i++ {
		apiObjects[i] = getFunc(rlv.Index(i))
	}
	return apiObjects
}

// Validate will return true if a list of resources is valid
func (v *Validator) Validate(namespace string, resourceList interface{}) error {
	apiObjects := toAPIObjectList(resourceList)

	names := map[string]bool{}
	for _, obj := range apiObjects {
		name := obj.GetName()
		if names[name] {
			return errors.Errorf("duplicate name %s for object %s", name, objID(obj))
		}

		if err := v.validateObject(namespace, obj); err != nil {
			return err
		}
		names[name] = true
	}
	return nil
}

// ValidateObject will return true if the object metadata for a resource is valid
func (v *Validator) ValidateObject(namespace string, obj metav1.Object) error {
	return v.validateObject(namespace, obj.(apiObject))
}

func (v *Validator) validateObject(namespace string, obj apiObject) error {
	if err := v.validateName(namespace, obj); err != nil {
		return errors.Wrapf(err, "invalid name on object %s", objID(obj))
	}

	if err := checkNomosPrefix(obj.GetLabels()); err != nil {
		return errors.Wrapf(err, "invalid metadata label on object %s", objID(obj))
	}

	if err := checkNomosPrefix(obj.GetAnnotations()); err != nil {
		return errors.Wrapf(err, "invalid metadata annotation on object %s", objID(obj))
	}

	return nil
}

func checkNomosPrefix(m map[string]string) error {
	for k := range m {
		if strings.HasPrefix(k, "nomos.dev/") {
			if k == v1.NamespaceSelectorAnnotationKey || k == v1.AnnotationKeyDeclarationPath {
				continue
			}
			return errors.Errorf("key %s contains nomos.dev/ prefix", k)
		}
	}
	return nil
}

func objID(obj apiObject) string {
	return fmt.Sprintf("%s %q", obj.GetObjectKind().GroupVersionKind(), obj.GetName())
}
