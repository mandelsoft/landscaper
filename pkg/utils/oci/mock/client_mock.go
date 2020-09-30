// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/landscaper/pkg/utils/oci (interfaces: Client)

// Package mock_oci is a generated GoMock package.
package mock_oci

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	io "io"
	reflect "reflect"
)

// MockClient is a mock of Client interface
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Fetch mocks base method
func (m *MockClient) Fetch(arg0 context.Context, arg1 string, arg2 v1.Descriptor, arg3 io.Writer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Fetch", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// Fetch indicates an expected call of Fetch
func (mr *MockClientMockRecorder) Fetch(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fetch", reflect.TypeOf((*MockClient)(nil).Fetch), arg0, arg1, arg2, arg3)
}

// GetManifest mocks base method
func (m *MockClient) GetManifest(arg0 context.Context, arg1 string) (*v1.Manifest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetManifest", arg0, arg1)
	ret0, _ := ret[0].(*v1.Manifest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetManifest indicates an expected call of GetManifest
func (mr *MockClientMockRecorder) GetManifest(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetManifest", reflect.TypeOf((*MockClient)(nil).GetManifest), arg0, arg1)
}

// PushManifest mocks base method
func (m *MockClient) PushManifest(arg0 context.Context, arg1 string, arg2 *v1.Manifest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PushManifest", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// PushManifest indicates an expected call of PushManifest
func (mr *MockClientMockRecorder) PushManifest(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PushManifest", reflect.TypeOf((*MockClient)(nil).PushManifest), arg0, arg1, arg2)
}
