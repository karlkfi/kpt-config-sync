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

package testing

import (
	"testing"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestResourceType is a pretty much empty object that is here to allow for testing generic action
// related things.
type TestResourceType struct {
	meta_v1.ObjectMeta
}

// NewTestResourceType creates a new test resource in a convenient manner.
func NewTestResourceType(namespace, name string) *TestResourceType {
	return &TestResourceType{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// TestResourceInterfaceImpl is an object that is provided for testcases if they need an instance
// of something that implements ResourceInterface
type TestResourceInterfaceImpl struct {
	ObjArg *TestResourceType

	// Return values
	CreateRet   *TestResourceType
	CreateErr   error
	GetRet      *TestResourceType
	GetErr      error
	UpdateRet   *TestResourceType
	UpdateErr   error
	DeleteErr   error
	EqualReturn bool
	ValuesRet   map[string]interface{}
	ValuesErr   error

	t *testing.T
}

func NewTestResourceInterfaceImpl(t *testing.T) *TestResourceInterfaceImpl {
	testResourceInstance := NewTestResourceType("default", "testresourcename")
	return &TestResourceInterfaceImpl{
		t:           t,
		ObjArg:      testResourceInstance,
		EqualReturn: true,
		CreateRet:   testResourceInstance,
		GetRet:      testResourceInstance,
		UpdateRet:   testResourceInstance,
		ValuesRet:   map[string]interface{}{},
	}
}

// // TestResourceInterfaceImpl implements ResourceInterface
// var _ ResourceInterface = &TestResourceInterfaceImpl{}

// Create implements ResourceInterface
func (s *TestResourceInterfaceImpl) Create(obj interface{}) (interface{}, error) {
	return s.CreateRet, s.CreateErr
}

// Delete implements ResourceInterface
func (s *TestResourceInterfaceImpl) Delete(obj interface{}) error {
	return s.DeleteErr
}

// Get implements ResourceInterface
func (s *TestResourceInterfaceImpl) Get(obj interface{}) (interface{}, error) {
	return s.GetRet, s.GetErr
}

// Update implements ResourceInterface
func (s *TestResourceInterfaceImpl) Update(oldObj interface{}, newObj interface{}) (interface{}, error) {
	return s.UpdateRet, s.UpdateErr
}

// Type implements ResourceInterface
func (s *TestResourceInterfaceImpl) Type() string {
	return "testresource"
}

// Name implements ResourceInterface
func (s *TestResourceInterfaceImpl) Name(obj interface{}) string {
	return obj.(*TestResourceType).Name
}

// Namespace implements ResourceInterface
func (s *TestResourceInterfaceImpl) Namespace(obj interface{}) string {
	return obj.(*TestResourceType).Namespace
}

// Equal implements ResourceInterface
func (s *TestResourceInterfaceImpl) Equal(lhsObj interface{}, rhsObj interface{}) bool {
	return s.EqualReturn
}

// Values implements ResourceInterface
func (s *TestResourceInterfaceImpl) Values(namespace string) (map[string]interface{}, error) {
	return s.ValuesRet, s.ValuesErr
}
