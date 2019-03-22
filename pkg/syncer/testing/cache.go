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

// Code generated by MockGen. DO NOT EDIT.
// Source: sigs.k8s.io/controller-runtime/pkg/cache (interfaces: Cache)

// Package testing is a generated GoMock package.
package testing

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	cache "k8s.io/client-go/tools/cache"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockCache is a mock of Cache interface
type MockCache struct {
	ctrl     *gomock.Controller
	recorder *MockCacheMockRecorder
}

// MockCacheMockRecorder is the mock recorder for MockCache
type MockCacheMockRecorder struct {
	mock *MockCache
}

// NewMockCache creates a new mock instance
func NewMockCache(ctrl *gomock.Controller) *MockCache {
	mock := &MockCache{ctrl: ctrl}
	mock.recorder = &MockCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCache) EXPECT() *MockCacheMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockCache) Get(arg0 context.Context, arg1 types.NamespacedName, arg2 runtime.Object) error {
	ret := m.ctrl.Call(m, "Get", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Get indicates an expected call of Get
func (mr *MockCacheMockRecorder) Get(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCache)(nil).Get), arg0, arg1, arg2)
}

// GetInformer mocks base method
func (m *MockCache) GetInformer(arg0 runtime.Object) (cache.SharedIndexInformer, error) {
	ret := m.ctrl.Call(m, "GetInformer", arg0)
	ret0, _ := ret[0].(cache.SharedIndexInformer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInformer indicates an expected call of GetInformer
func (mr *MockCacheMockRecorder) GetInformer(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInformer", reflect.TypeOf((*MockCache)(nil).GetInformer), arg0)
}

// GetInformerForKind mocks base method
func (m *MockCache) GetInformerForKind(arg0 schema.GroupVersionKind) (cache.SharedIndexInformer, error) {
	ret := m.ctrl.Call(m, "GetInformerForKind", arg0)
	ret0, _ := ret[0].(cache.SharedIndexInformer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInformerForKind indicates an expected call of GetInformerForKind
func (mr *MockCacheMockRecorder) GetInformerForKind(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInformerForKind", reflect.TypeOf((*MockCache)(nil).GetInformerForKind), arg0)
}

// IndexField mocks base method
func (m *MockCache) IndexField(arg0 runtime.Object, arg1 string, arg2 client.IndexerFunc) error {
	ret := m.ctrl.Call(m, "IndexField", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// IndexField indicates an expected call of IndexField
func (mr *MockCacheMockRecorder) IndexField(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IndexField", reflect.TypeOf((*MockCache)(nil).IndexField), arg0, arg1, arg2)
}

// List mocks base method
func (m *MockCache) List(arg0 context.Context, arg1 *client.ListOptions, arg2 runtime.Object) error {
	ret := m.ctrl.Call(m, "List", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// List indicates an expected call of List
func (mr *MockCacheMockRecorder) List(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockCache)(nil).List), arg0, arg1, arg2)
}

// Start mocks base method
func (m *MockCache) Start(arg0 <-chan struct{}) error {
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (mr *MockCacheMockRecorder) Start(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockCache)(nil).Start), arg0)
}

// WaitForCacheSync mocks base method
func (m *MockCache) WaitForCacheSync(arg0 <-chan struct{}) bool {
	ret := m.ctrl.Call(m, "WaitForCacheSync", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// WaitForCacheSync indicates an expected call of WaitForCacheSync
func (mr *MockCacheMockRecorder) WaitForCacheSync(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForCacheSync", reflect.TypeOf((*MockCache)(nil).WaitForCacheSync), arg0)
}
