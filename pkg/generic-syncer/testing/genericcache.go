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

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/google/nomos/pkg/generic-syncer/cache (interfaces: GenericCache)

// Package testing is a generated GoMock package.
package testing

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	cache "k8s.io/client-go/tools/cache"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockGenericCache is a mock of GenericCache interface
type MockGenericCache struct {
	ctrl     *gomock.Controller
	recorder *MockGenericCacheMockRecorder
}

// MockGenericCacheMockRecorder is the mock recorder for MockGenericCache
type MockGenericCacheMockRecorder struct {
	mock *MockGenericCache
}

// NewMockGenericCache creates a new mock instance
func NewMockGenericCache(ctrl *gomock.Controller) *MockGenericCache {
	mock := &MockGenericCache{ctrl: ctrl}
	mock.recorder = &MockGenericCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockGenericCache) EXPECT() *MockGenericCacheMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockGenericCache) Get(arg0 context.Context, arg1 types.NamespacedName, arg2 runtime.Object) error {
	ret := m.ctrl.Call(m, "Get", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Get indicates an expected call of Get
func (mr *MockGenericCacheMockRecorder) Get(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockGenericCache)(nil).Get), arg0, arg1, arg2)
}

// GetInformer mocks base method
func (m *MockGenericCache) GetInformer(arg0 runtime.Object) (cache.SharedIndexInformer, error) {
	ret := m.ctrl.Call(m, "GetInformer", arg0)
	ret0, _ := ret[0].(cache.SharedIndexInformer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInformer indicates an expected call of GetInformer
func (mr *MockGenericCacheMockRecorder) GetInformer(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInformer", reflect.TypeOf((*MockGenericCache)(nil).GetInformer), arg0)
}

// GetInformerForKind mocks base method
func (m *MockGenericCache) GetInformerForKind(arg0 schema.GroupVersionKind) (cache.SharedIndexInformer, error) {
	ret := m.ctrl.Call(m, "GetInformerForKind", arg0)
	ret0, _ := ret[0].(cache.SharedIndexInformer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInformerForKind indicates an expected call of GetInformerForKind
func (mr *MockGenericCacheMockRecorder) GetInformerForKind(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInformerForKind", reflect.TypeOf((*MockGenericCache)(nil).GetInformerForKind), arg0)
}

// IndexField mocks base method
func (m *MockGenericCache) IndexField(arg0 runtime.Object, arg1 string, arg2 client.IndexerFunc) error {
	ret := m.ctrl.Call(m, "IndexField", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// IndexField indicates an expected call of IndexField
func (mr *MockGenericCacheMockRecorder) IndexField(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IndexField", reflect.TypeOf((*MockGenericCache)(nil).IndexField), arg0, arg1, arg2)
}

// List mocks base method
func (m *MockGenericCache) List(arg0 context.Context, arg1 *client.ListOptions, arg2 runtime.Object) error {
	ret := m.ctrl.Call(m, "List", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// List indicates an expected call of List
func (mr *MockGenericCacheMockRecorder) List(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockGenericCache)(nil).List), arg0, arg1, arg2)
}

// Start mocks base method
func (m *MockGenericCache) Start(arg0 <-chan struct{}) error {
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (mr *MockGenericCacheMockRecorder) Start(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockGenericCache)(nil).Start), arg0)
}

// UnstructuredList mocks base method
func (m *MockGenericCache) UnstructuredList(arg0 schema.GroupVersionKind) ([]*unstructured.Unstructured, error) {
	ret := m.ctrl.Call(m, "UnstructuredList", arg0)
	ret0, _ := ret[0].([]*unstructured.Unstructured)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnstructuredList indicates an expected call of UnstructuredList
func (mr *MockGenericCacheMockRecorder) UnstructuredList(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnstructuredList", reflect.TypeOf((*MockGenericCache)(nil).UnstructuredList), arg0)
}

// WaitForCacheSync mocks base method
func (m *MockGenericCache) WaitForCacheSync(arg0 <-chan struct{}) bool {
	ret := m.ctrl.Call(m, "WaitForCacheSync", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// WaitForCacheSync indicates an expected call of WaitForCacheSync
func (mr *MockGenericCacheMockRecorder) WaitForCacheSync(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForCacheSync", reflect.TypeOf((*MockGenericCache)(nil).WaitForCacheSync), arg0)
}
