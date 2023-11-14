// Code generated by MockGen. DO NOT EDIT.
// Source: handlers_constructor.go
//
// Generated by this command:
//
//	mockgen -package=mocks -destination=internal/mocks/handlers_constructor_mock.go -source=handlers_constructor.go -mock_names=HandlersConstructorI=MockHandlersConstructor
//
// Package mocks is a generated GoMock package.
package mocks

import (
	slog "log/slog"
	exec "os/exec"
	reflect "reflect"

	handlers "github.com/k-lb/entrypoint-framework/handlers"
	gomock "go.uber.org/mock/gomock"
)

// MockHandlersConstructorIface is a mock of HandlersConstructorIface interface.
type MockHandlersConstructorIface struct {
	ctrl     *gomock.Controller
	recorder *MockHandlersConstructorIfaceMockRecorder
}

// MockHandlersConstructorIfaceMockRecorder is the mock recorder for MockHandlersConstructorIface.
type MockHandlersConstructorIfaceMockRecorder struct {
	mock *MockHandlersConstructorIface
}

// NewMockHandlersConstructorIface creates a new mock instance.
func NewMockHandlersConstructorIface(ctrl *gomock.Controller) *MockHandlersConstructorIface {
	mock := &MockHandlersConstructorIface{ctrl: ctrl}
	mock.recorder = &MockHandlersConstructorIfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHandlersConstructorIface) EXPECT() *MockHandlersConstructorIfaceMockRecorder {
	return m.recorder
}

// NewActivationHandler mocks base method.
func (m *MockHandlersConstructorIface) NewActivationHandler(activationFile string, logger *slog.Logger) (handlers.ActivationHandler, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewActivationHandler", activationFile, logger)
	ret0, _ := ret[0].(handlers.ActivationHandler)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewActivationHandler indicates an expected call of NewActivationHandler.
func (mr *MockHandlersConstructorIfaceMockRecorder) NewActivationHandler(activationFile, logger any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewActivationHandler", reflect.TypeOf((*MockHandlersConstructorIface)(nil).NewActivationHandler), activationFile, logger)
}

// NewConfigurationHandler mocks base method.
func (m *MockHandlersConstructorIface) NewConfigurationHandler(newConfigFile, newConfigDir, oldConfigDir string, logger *slog.Logger) (handlers.ConfigurationHandler[handlers.UpdateResult], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewConfigurationHandler", newConfigFile, newConfigDir, oldConfigDir, logger)
	ret0, _ := ret[0].(handlers.ConfigurationHandler[handlers.UpdateResult])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewConfigurationHandler indicates an expected call of NewConfigurationHandler.
func (mr *MockHandlersConstructorIfaceMockRecorder) NewConfigurationHandler(newConfigFile, newConfigDir, oldConfigDir, logger any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewConfigurationHandler", reflect.TypeOf((*MockHandlersConstructorIface)(nil).NewConfigurationHandler), newConfigFile, newConfigDir, oldConfigDir, logger)
}

// NewProcessHandler mocks base method.
func (m *MockHandlersConstructorIface) NewProcessHandler(cmd *exec.Cmd, logger *slog.Logger) (handlers.ProcessHandler, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewProcessHandler", cmd, logger)
	ret0, _ := ret[0].(handlers.ProcessHandler)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewProcessHandler indicates an expected call of NewProcessHandler.
func (mr *MockHandlersConstructorIfaceMockRecorder) NewProcessHandler(cmd, logger any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewProcessHandler", reflect.TypeOf((*MockHandlersConstructorIface)(nil).NewProcessHandler), cmd, logger)
}