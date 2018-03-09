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

package actions

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/client/action"
	"github.com/pkg/errors"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceInterface is here to deal with all the actions being roughly equivalent with the
// exception of a few places.  The main goal of this is to reduce the surface area required for
// performing action tests.
// NOTE: All "interface{}" types passed to this function will be a non-nil pointer to the instance.
type ResourceInterface interface {
	// Create takes an object, then calls the k8s create API for the type and returns the return values
	// from the Create method
	Create(obj interface{}) (interface{}, error)
	// Delete takes an object, calls the delete API call for the type then returns the error value
	Delete(obj interface{}) error
	// Get takes an object, calls the Get API then returns the resource and error
	Get(obj interface{}) (interface{}, error)
	// Update takes an old object and a new object
	Update(oldObj interface{}, newObj interface{}) (interface{}, error)
	// Type returns the name of the type of resource this operates on
	Type() string
	// Name returns the name of the resource in obj
	Name(obj interface{}) string
	// Namespace returns the namespace of the resource in obj
	Namespace(obj interface{}) string
	// Equal returns true if the effective contents of the two objects are equal.
	Equal(lhs interface{}, rhs interface{}) bool
	// Values returns a map of name to value for all resources that currently exist in the
	// namespace. If the resource is cluster level, this should return always return the cluster
	// resource values.
	Values(namespace string) (map[string]interface{}, error)
}

type genericActionBase struct {
	operation         string
	resource          interface{}
	resourceInterface ResourceInterface
}

// MetaEquals returns equals for the object meta field when update is concerned.
func MetaEquals(lhs meta_v1.ObjectMeta, rhs meta_v1.ObjectMeta) bool {
	return reflect.DeepEqual(lhs.Labels, rhs.Labels) &&
		reflect.DeepEqual(lhs.Annotations, rhs.Annotations)
}

// Resource implements Interface
func (s *genericActionBase) Resource() string {
	return s.resourceInterface.Type()
}

// Namespace implements Interface
func (s *genericActionBase) Namespace() string {
	return s.resourceInterface.Namespace(s.resource)
}

// Name returns the name of the resource
func (s *genericActionBase) Name() string {
	return s.resourceInterface.Name(s.resource)
}

// Operation implements Interface
func (s *genericActionBase) Operation() string {
	return s.operation
}

// String implements Interface
func (s *genericActionBase) String() string {
	namespace := s.Namespace()
	if namespace != "" {
		return fmt.Sprintf(
			"%s.%s.%s.%s",
			s.Resource(),
			namespace,
			s.Name(),
			s.Operation())
	}
	return fmt.Sprintf(
		"%s.%s.%s",
		s.Resource(),
		s.Name(),
		s.Operation())
}

// GenericDeleteAction implements deleting a resource with an associated resourceInterface.
type GenericDeleteAction struct {
	genericActionBase
}

var _ action.Interface = &GenericDeleteAction{}

// NewGenericDeleteAction creates a delete action from a resource and resource interface.
func NewGenericDeleteAction(
	resource interface{},
	resourceInterface ResourceInterface) *GenericDeleteAction {
	return &GenericDeleteAction{
		genericActionBase: genericActionBase{
			operation:         "delete",
			resource:          resource,
			resourceInterface: resourceInterface,
		},
	}
}

// Execute implements Interface
func (s *GenericDeleteAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)
	_, err := s.resourceInterface.Get(s.resource)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "Get failed for %s", s)
	}

	err = s.resourceInterface.Delete(s.resource)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "Delete failed for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}

// GenericUpsertAction implements upsert on top of the action base
type GenericUpsertAction struct {
	genericActionBase
}

var _ action.Interface = &GenericUpsertAction{}

// NewGenericUpsertAction creates an upsert action from a resource (a kubernetes API type) and
// a resource interface.
func NewGenericUpsertAction(
	resource interface{},
	resourceInterface ResourceInterface) *GenericUpsertAction {
	action := &GenericUpsertAction{
		genericActionBase: genericActionBase{
			operation:         "upsert",
			resource:          resource,
			resourceInterface: resourceInterface,
		},
	}
	glog.V(10).Infof("New upsert action: %v", action)
	return action
}

func (s *GenericUpsertAction) create() error {
	if _, err := s.resourceInterface.Create(s.resource); err != nil {
		if api_errors.IsAlreadyExists(err) {
			return s.upsert()
		}
		return errors.Wrapf(err, "Failed during create for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}

func (s *GenericUpsertAction) upsert() error {
	resouce, err := s.resourceInterface.Get(s.resource)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return s.create()
		}
		return errors.Wrapf(err, "Failed to get resource for %s", s)
	}

	if s.resourceInterface.Equal(s.resource, resouce) {
		return nil
	}

	if _, err = s.resourceInterface.Update(resouce, s.resource); err != nil {
		return errors.Wrapf(err, "Failed to update for %s", s)
	}
	glog.V(1).Infof("OK: %s", s)
	return nil
}

// Execute implements Interface
func (s *GenericUpsertAction) Execute() error {
	glog.V(1).Infof("Executing %s", s)
	return s.upsert()
}
