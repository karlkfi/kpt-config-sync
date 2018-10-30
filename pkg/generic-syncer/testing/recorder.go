// Code generated by MockGen. DO NOT EDIT.
// Source: k8s.io/client-go/tools/record (interfaces: EventRecorder)

// Package testing is a generated GoMock package.
package testing

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// MockEventRecorder is a mock of EventRecorder interface
type MockEventRecorder struct {
	ctrl     *gomock.Controller
	recorder *MockEventRecorderMockRecorder
}

// MockEventRecorderMockRecorder is the mock recorder for MockEventRecorder
type MockEventRecorderMockRecorder struct {
	mock *MockEventRecorder
}

// NewMockEventRecorder creates a new mock instance
func NewMockEventRecorder(ctrl *gomock.Controller) *MockEventRecorder {
	mock := &MockEventRecorder{ctrl: ctrl}
	mock.recorder = &MockEventRecorderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockEventRecorder) EXPECT() *MockEventRecorderMockRecorder {
	return m.recorder
}

// AnnotatedEventf mocks base method
func (m *MockEventRecorder) AnnotatedEventf(arg0 runtime.Object, arg1 map[string]string, arg2, arg3, arg4 string, arg5 ...interface{}) {
	varargs := []interface{}{arg0, arg1, arg2, arg3, arg4}
	for _, a := range arg5 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "AnnotatedEventf", varargs...)
}

// AnnotatedEventf indicates an expected call of AnnotatedEventf
func (mr *MockEventRecorderMockRecorder) AnnotatedEventf(arg0, arg1, arg2, arg3, arg4 interface{}, arg5 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1, arg2, arg3, arg4}, arg5...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AnnotatedEventf", reflect.TypeOf((*MockEventRecorder)(nil).AnnotatedEventf), varargs...)
}

// Event mocks base method
func (m *MockEventRecorder) Event(arg0 runtime.Object, arg1, arg2, arg3 string) {
	m.ctrl.Call(m, "Event", arg0, arg1, arg2, arg3)
}

// Event indicates an expected call of Event
func (mr *MockEventRecorderMockRecorder) Event(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Event", reflect.TypeOf((*MockEventRecorder)(nil).Event), arg0, arg1, arg2, arg3)
}

// Eventf mocks base method
func (m *MockEventRecorder) Eventf(arg0 runtime.Object, arg1, arg2, arg3 string, arg4 ...interface{}) {
	varargs := []interface{}{arg0, arg1, arg2, arg3}
	for _, a := range arg4 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Eventf", varargs...)
}

// Eventf indicates an expected call of Eventf
func (mr *MockEventRecorderMockRecorder) Eventf(arg0, arg1, arg2, arg3 interface{}, arg4 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1, arg2, arg3}, arg4...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Eventf", reflect.TypeOf((*MockEventRecorder)(nil).Eventf), varargs...)
}

// PastEventf mocks base method
func (m *MockEventRecorder) PastEventf(arg0 runtime.Object, arg1 v1.Time, arg2, arg3, arg4 string, arg5 ...interface{}) {
	varargs := []interface{}{arg0, arg1, arg2, arg3, arg4}
	for _, a := range arg5 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "PastEventf", varargs...)
}

// PastEventf indicates an expected call of PastEventf
func (mr *MockEventRecorderMockRecorder) PastEventf(arg0, arg1, arg2, arg3, arg4 interface{}, arg5 ...interface{}) *gomock.Call {
	varargs := append([]interface{}{arg0, arg1, arg2, arg3, arg4}, arg5...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PastEventf", reflect.TypeOf((*MockEventRecorder)(nil).PastEventf), varargs...)
}
