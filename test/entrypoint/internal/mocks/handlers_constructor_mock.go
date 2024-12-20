// Code generated by MockGen. DO NOT EDIT.
// Source: handlers_constructor.go
//
// Generated by this command:
//
//	mockgen -package=mocks -destination=internal/mocks/handlers_constructor_mock.go -source=handlers_constructor.go -mock_names=HandlersConstructorIface=MockHandlersConstructor
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

// MockHandlersConstructor is a mock of HandlersConstructorIface interface.
type MockHandlersConstructor struct {
	ctrl     *gomock.Controller
	recorder *MockHandlersConstructorMockRecorder
	isgomock struct{}
}

// MockHandlersConstructorMockRecorder is the mock recorder for MockHandlersConstructor.
type MockHandlersConstructorMockRecorder struct {
	mock *MockHandlersConstructor
}

// NewMockHandlersConstructor creates a new mock instance.
func NewMockHandlersConstructor(ctrl *gomock.Controller) *MockHandlersConstructor {
	mock := &MockHandlersConstructor{ctrl: ctrl}
	mock.recorder = &MockHandlersConstructorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockHandlersConstructor) EXPECT() *MockHandlersConstructorMockRecorder {
	return m.recorder
}

// NewActivationHandler mocks base method.
func (m *MockHandlersConstructor) NewActivationHandler(activationFile string, logger *slog.Logger) (handlers.ActivationHandler, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewActivationHandler", activationFile, logger)
	ret0, _ := ret[0].(handlers.ActivationHandler)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewActivationHandler indicates an expected call of NewActivationHandler.
func (mr *MockHandlersConstructorMockRecorder) NewActivationHandler(activationFile, logger any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewActivationHandler", reflect.TypeOf((*MockHandlersConstructor)(nil).NewActivationHandler), activationFile, logger)
}

// NewConfigurationHandler mocks base method.
func (m *MockHandlersConstructor) NewConfigurationHandler(newConfigFile, newConfigDir, oldConfigDir string, logger *slog.Logger) (handlers.ConfigurationHandler[handlers.UpdateResult], error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewConfigurationHandler", newConfigFile, newConfigDir, oldConfigDir, logger)
	ret0, _ := ret[0].(handlers.ConfigurationHandler[handlers.UpdateResult])
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewConfigurationHandler indicates an expected call of NewConfigurationHandler.
func (mr *MockHandlersConstructorMockRecorder) NewConfigurationHandler(newConfigFile, newConfigDir, oldConfigDir, logger any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewConfigurationHandler", reflect.TypeOf((*MockHandlersConstructor)(nil).NewConfigurationHandler), newConfigFile, newConfigDir, oldConfigDir, logger)
}

// NewProcessHandler mocks base method.
func (m *MockHandlersConstructor) NewProcessHandler(cmd *exec.Cmd, logger *slog.Logger) (handlers.ProcessHandler, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewProcessHandler", cmd, logger)
	ret0, _ := ret[0].(handlers.ProcessHandler)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewProcessHandler indicates an expected call of NewProcessHandler.
func (mr *MockHandlersConstructorMockRecorder) NewProcessHandler(cmd, logger any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewProcessHandler", reflect.TypeOf((*MockHandlersConstructor)(nil).NewProcessHandler), cmd, logger)
}
