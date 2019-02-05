// Code generated by MockGen. DO NOT EDIT.
// Source: k8s.io/client-go/discovery (interfaces: DiscoveryInterface)

// Package testing is a generated GoMock package.
package testing

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	OpenAPIv2 "github.com/googleapis/gnostic/OpenAPIv2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	version "k8s.io/apimachinery/pkg/version"
	rest "k8s.io/client-go/rest"
)

// MockDiscoveryInterface is a mock of DiscoveryInterface interface
type MockDiscoveryInterface struct {
	ctrl     *gomock.Controller
	recorder *MockDiscoveryInterfaceMockRecorder
}

// MockDiscoveryInterfaceMockRecorder is the mock recorder for MockDiscoveryInterface
type MockDiscoveryInterfaceMockRecorder struct {
	mock *MockDiscoveryInterface
}

// NewMockDiscoveryInterface creates a new mock instance
func NewMockDiscoveryInterface(ctrl *gomock.Controller) *MockDiscoveryInterface {
	mock := &MockDiscoveryInterface{ctrl: ctrl}
	mock.recorder = &MockDiscoveryInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDiscoveryInterface) EXPECT() *MockDiscoveryInterfaceMockRecorder {
	return m.recorder
}

// OpenAPISchema mocks base method
func (m *MockDiscoveryInterface) OpenAPISchema() (*OpenAPIv2.Document, error) {
	ret := m.ctrl.Call(m, "OpenAPISchema")
	ret0, _ := ret[0].(*OpenAPIv2.Document)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// OpenAPISchema indicates an expected call of OpenAPISchema
func (mr *MockDiscoveryInterfaceMockRecorder) OpenAPISchema() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OpenAPISchema", reflect.TypeOf((*MockDiscoveryInterface)(nil).OpenAPISchema))
}

// RESTClient mocks base method
func (m *MockDiscoveryInterface) RESTClient() rest.Interface {
	ret := m.ctrl.Call(m, "RESTClient")
	ret0, _ := ret[0].(rest.Interface)
	return ret0
}

// RESTClient indicates an expected call of RESTClient
func (mr *MockDiscoveryInterfaceMockRecorder) RESTClient() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RESTClient", reflect.TypeOf((*MockDiscoveryInterface)(nil).RESTClient))
}

// ServerGroups mocks base method
func (m *MockDiscoveryInterface) ServerGroups() (*v1.APIGroupList, error) {
	ret := m.ctrl.Call(m, "ServerGroups")
	ret0, _ := ret[0].(*v1.APIGroupList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerGroups indicates an expected call of ServerGroups
func (mr *MockDiscoveryInterfaceMockRecorder) ServerGroups() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerGroups", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerGroups))
}

// ServerPreferredNamespacedResources mocks base method
func (m *MockDiscoveryInterface) ServerPreferredNamespacedResources() ([]*v1.APIResourceList, error) {
	ret := m.ctrl.Call(m, "ServerPreferredNamespacedResources")
	ret0, _ := ret[0].([]*v1.APIResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerPreferredNamespacedResources indicates an expected call of ServerPreferredNamespacedResources
func (mr *MockDiscoveryInterfaceMockRecorder) ServerPreferredNamespacedResources() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerPreferredNamespacedResources", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerPreferredNamespacedResources))
}

// ServerPreferredResources mocks base method
func (m *MockDiscoveryInterface) ServerPreferredResources() ([]*v1.APIResourceList, error) {
	ret := m.ctrl.Call(m, "ServerPreferredResources")
	ret0, _ := ret[0].([]*v1.APIResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerPreferredResources indicates an expected call of ServerPreferredResources
func (mr *MockDiscoveryInterfaceMockRecorder) ServerPreferredResources() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerPreferredResources", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerPreferredResources))
}

// ServerResources mocks base method
func (m *MockDiscoveryInterface) ServerResources() ([]*v1.APIResourceList, error) {
	ret := m.ctrl.Call(m, "ServerResources")
	ret0, _ := ret[0].([]*v1.APIResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerResources indicates an expected call of ServerResources
func (mr *MockDiscoveryInterfaceMockRecorder) ServerResources() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerResources", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerResources))
}

// ServerResourcesForGroupVersion mocks base method
func (m *MockDiscoveryInterface) ServerResourcesForGroupVersion(arg0 string) (*v1.APIResourceList, error) {
	ret := m.ctrl.Call(m, "ServerResourcesForGroupVersion", arg0)
	ret0, _ := ret[0].(*v1.APIResourceList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerResourcesForGroupVersion indicates an expected call of ServerResourcesForGroupVersion
func (mr *MockDiscoveryInterfaceMockRecorder) ServerResourcesForGroupVersion(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerResourcesForGroupVersion", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerResourcesForGroupVersion), arg0)
}

// ServerVersion mocks base method
func (m *MockDiscoveryInterface) ServerVersion() (*version.Info, error) {
	ret := m.ctrl.Call(m, "ServerVersion")
	ret0, _ := ret[0].(*version.Info)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerVersion indicates an expected call of ServerVersion
func (mr *MockDiscoveryInterfaceMockRecorder) ServerVersion() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerVersion", reflect.TypeOf((*MockDiscoveryInterface)(nil).ServerVersion))
}
