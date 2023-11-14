/*
 *  Copyright (c) 2023 Samsung Electronics Co., Ltd All Rights Reserved
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License
 */

package main

import (
	"bytes"
	"errors"
	"os/exec"

	m "go.uber.org/mock/gomock"

	"github.com/k-lb/entrypoint-framework/handlers"
)

func sliceToChan[T any](slice []T) <-chan T {
	c := make(chan T, len(slice))
	for _, val := range slice {
		c <- val
	}
	return c
}

func (e *EntrypointTestSuite) TestEntrypointCmd() {
	e.Run("should return expected value", func() {
		e.T().Parallel()
		e.Equal(exec.Command("sleep", "1"), cmd())
	})
}

func (e *EntrypointTestSuite) TestEntrypointInitialization() {
	testCases := [...]struct {
		name                                              string
		activationError, configurationError, processError error
	}{
		{name: "when can't create activation handler, should return an error", activationError: errors.New("create activation handler error")},
		{name: "when can't create configuration handler, should return an error", configurationError: errors.New("create configuration handler error")},
		{name: "when can't create process handler, should return an error", processError: errors.New("create process handler error")},
		{name: "when nothing returns any errors, should create all handlers, initialize state and return no error"},
	}
	for _, test := range testCases {
		test := test
		e.runWithMockEntrypoint(test.name, func(entrypoint *Entrypoint, mocks *mocksControl, _ *bytes.Buffer) {
			expectedError := func() error {
				mocks.hc.EXPECT().NewActivationHandler(watchedActivationPath, entrypoint.log).Times(1).
					Return(mocks.activation, test.activationError)
				if test.activationError != nil {
					return test.activationError
				}
				mocks.hc.EXPECT().
					NewConfigurationHandler(watchedConfigurationPath, newConfigurationDir, oldConfigurationDir, entrypoint.log).
					Times(1).Return(mocks.configuration, test.configurationError)
				if test.configurationError != nil {
					return test.configurationError
				}
				mocks.hc.EXPECT().NewProcessHandler(m.Any(), entrypoint.log).Times(1).Return(mocks.process, test.processError)
				return test.processError
			}()
			err := entrypoint.initialize()

			e.ErrorIs(err, expectedError)
			if expectedError == nil {
				e.Zero(entrypoint.configUpdatesRunning)
				e.NotNil(entrypoint.activation)
				e.NotNil(entrypoint.configuration)
				e.NotNil(entrypoint.process)
				e.Equal(State{inactive, notReady, dead}, entrypoint.state)
			}
		})
	}
}

func (e *EntrypointTestSuite) TestEntrypointTearDown() {
	e.runWithMockEntrypoint("should close all handlers", func(entrypoint *Entrypoint, mocks *mocksControl, _ *bytes.Buffer) {
		mocks.activation.EXPECT().Close().Times(1)
		mocks.configuration.EXPECT().Close().Times(1)
		mocks.process.EXPECT().Kill().Return(nil).Times(1)
		entrypoint.tearDown()
	})
}

func (e *EntrypointTestSuite) TestEntrypointChangingStateByEvents() {
	testCases := [...]struct {
		name string

		activationWasChanged []handlers.ActivationEvent

		configurationWasChanged                            []error
		configurationUpdateResult                          []handlers.UpdateResult
		wasConfigChanged, expectedWasConfigChanged         bool
		configUpdatesRunning, expectedConfigUpdatesRunning int

		processStarted []error
		processEnded   []error

		initialState  State
		expectedState State
		logContains   string
	}{
		{name: "When activation was changed to active without any errors and a state is inactive, should change the state to active",
			activationWasChanged: []handlers.ActivationEvent{{State: true}},
			initialState:         State{activation: inactive},
			expectedState:        State{activation: active}},
		{name: "When activation was changed to inactive without any errors and a state is active, should change the state to inactive",
			activationWasChanged: []handlers.ActivationEvent{{State: false}},
			initialState:         State{activation: active},
			expectedState:        State{activation: inactive}},
		{name: "When activation was changed to active with an error and a state is inactive, shouldn't change the state and log the error",
			activationWasChanged: []handlers.ActivationEvent{{State: true, Error: errors.New("activation error")}},
			initialState:         State{activation: inactive},
			expectedState:        State{activation: inactive},
			logContains:          "activation error"},
		{name: "When configuration was changed without any errors and a state is notReady, should change the state to changed",
			configurationWasChanged: []error{nil}, initialState: State{configuration: notReady},
			expectedState: State{configuration: changed}},
		{name: "When configuration was changed with an error and a state is notReady, shouldn't change the state and log the error",
			configurationWasChanged: []error{errors.New("config change error")},
			initialState:            State{configuration: notReady},
			expectedState:           State{configuration: notReady},
			logContains:             "config change error"},
		{name: "When configuration was updated without any errors and files were changed, should change wasConfigChanged to true",
			configurationUpdateResult: []handlers.UpdateResult{{ChangedFiles: map[string]handlers.Modification{"test_file": handlers.Created}}},
			wasConfigChanged:          false, expectedWasConfigChanged: true,
			configUpdatesRunning: 2, expectedConfigUpdatesRunning: 1,
			logContains: "File test_file was created"},
		{name: "When configuration was updated without any errors and no files were changed, shouldn't change wasConfigChanged",
			configurationUpdateResult: []handlers.UpdateResult{{}},
			wasConfigChanged:          true, expectedWasConfigChanged: true,
			configUpdatesRunning: 2, expectedConfigUpdatesRunning: 1},
		{name: "When configuration was updated without any errors, wasConfigChanged is true and configUpdatesRunning equals 0, should change the state to updated",
			configurationUpdateResult: []handlers.UpdateResult{{}},
			initialState:              State{configuration: notReady},
			expectedState:             State{configuration: updated},
			wasConfigChanged:          true, expectedWasConfigChanged: true,
			configUpdatesRunning: 1, expectedConfigUpdatesRunning: 0},
		{name: "When configuration was updated without any errors, wasConfigChanged is false and configUpdatesRunning equals 0, should change the state to applied",
			configurationUpdateResult: []handlers.UpdateResult{{}},
			initialState:              State{configuration: notReady},
			expectedState:             State{configuration: applied},
			wasConfigChanged:          false, expectedWasConfigChanged: false,
			configUpdatesRunning: 1, expectedConfigUpdatesRunning: 0},
		{name: "When process was started without any errors and a state is dead, should change the state to alive",
			processStarted: []error{nil},
			initialState:   State{process: dead},
			expectedState:  State{process: alive}},
		{name: "When process was started without any errors and a state is dead and updated, should change the state to alive and applied",
			processStarted:   []error{nil},
			initialState:     State{configuration: updated, process: dead},
			expectedState:    State{configuration: applied, process: alive},
			wasConfigChanged: true, expectedWasConfigChanged: false},
		{name: "When process was started with an error and a state is dead, shouldn't change the state and log the error",
			processStarted: []error{errors.New("process started error")},
			initialState:   State{process: dead},
			expectedState:  State{process: dead},
			logContains:    "process started error"},
		{name: "When process was ended without any errors and a state is alive, should change the state to dead",
			processEnded:  []error{nil},
			initialState:  State{process: alive},
			expectedState: State{process: dead}},
		{name: "When process was ended with an error and a state is alive, should change the state to dead and log the error",
			processEnded:  []error{errors.New("process ended error")},
			initialState:  State{process: alive},
			expectedState: State{process: dead},
			logContains:   "process ended error"},
	}
	for _, test := range testCases {
		test := test
		e.runWithMockEntrypoint(test.name, func(entrypoint *Entrypoint, mocks *mocksControl, logBuf *bytes.Buffer) {
			mocks.activation.EXPECT().GetWasChangedChannel().Return(sliceToChan(test.activationWasChanged)).Times(1)
			mocks.configuration.EXPECT().GetWasChangedChannel().Return(sliceToChan(test.configurationWasChanged)).Times(1)
			mocks.configuration.EXPECT().GetUpdateResultChannel().Return(sliceToChan(test.configurationUpdateResult)).Times(1)
			mocks.process.EXPECT().GetStartedChannel().Return(sliceToChan(test.processStarted)).Times(1)
			mocks.process.EXPECT().GetEndedChannel().Return(sliceToChan(test.processEnded)).Times(1)
			entrypoint.state = test.initialState
			entrypoint.wasConfigChanged = test.wasConfigChanged
			entrypoint.configUpdatesRunning = test.configUpdatesRunning
			entrypoint.changeStateByEvent()

			e.Equal(test.expectedState, entrypoint.state)
			e.Equal(test.expectedWasConfigChanged, entrypoint.wasConfigChanged)
			e.Equal(test.expectedConfigUpdatesRunning, entrypoint.configUpdatesRunning)
			e.Contains(logBuf.String(), test.logContains)
		})
	}
}

func (e *EntrypointTestSuite) TestEntrypointHandlingStatusChanged() {
	startTestCases := [...]struct {
		name                 string
		state                State
		errNewProcessHandler error
	}{
		{name: "When state is active, applied, dead, should create a new process handler, start it and change process state to changing",
			state: State{active, applied, dead}},
		{name: "When state is active, applied, dead and NewProcessHandler returns an error, should create a new process handler and change process state to changing",
			state:                State{active, applied, dead},
			errNewProcessHandler: errors.New("new process handler error")},
		{name: "When state is active, updated, dead, should create a new process handler, start it and change process state to changing",
			state: State{active, updated, dead}},
		{name: "When state is active, updated, dead and NewProcessHandler returns an error, should create a new process handler and change process state to changing",
			state:                State{active, updated, dead},
			errNewProcessHandler: errors.New("new process handler error")},
	}
	for _, test := range startTestCases {
		test := test
		e.runWithMockEntrypoint(test.name, func(entrypoint *Entrypoint, mocks *mocksControl, logBuf *bytes.Buffer) {
			entrypoint.state = test.state
			mocks.hc.EXPECT().NewProcessHandler(m.Any(), entrypoint.log).
				Return(mocks.process, test.errNewProcessHandler).Times(1)
			if test.errNewProcessHandler == nil {
				mocks.process.EXPECT().Start().Times(1)
			}
			entrypoint.process = nil
			entrypoint.handleStatusChange()

			if test.errNewProcessHandler == nil {
				test.state.process = changing
			} else {
				e.Contains(logBuf.String(), "could not start an entrypoint")
			}
			e.NotNil(entrypoint.process)
			e.Equal(test.state, entrypoint.state)
		})
	}

	restartTestCases := [...]struct {
		name                          string
		state                         State
		errNewProcessHandler, errKill error
		logContains                   string
	}{
		{name: "When state is active, updated, alive and killing returns an error, should try killing the process and log error",
			state:       State{active, updated, alive},
			errKill:     errors.New("signal error"),
			logContains: "signal error"},
		{name: "When state is active, updated, alive and NewProcessHandler returns an error, should kill the process, change process state to changing, try creating a new process handler and log error",
			state:                State{active, updated, alive},
			errNewProcessHandler: errors.New("new process handler error"),
			logContains:          "new process handler error"},
		{name: "When state is active, updated, alive, should kill the process, change process state to changing, create a new process handler and starts it",
			state: State{active, updated, alive}},
	}
	for _, test := range restartTestCases {
		test := test
		e.runWithMockEntrypoint(test.name, func(entrypoint *Entrypoint, mocks *mocksControl, logBuf *bytes.Buffer) {
			entrypoint.state = test.state
			expectedError := func() error {
				if mocks.process.EXPECT().Kill().Return(test.errKill).Times(1); test.errKill != nil {
					return test.errKill
				}
				test.state.process = changing
				mocks.hc.EXPECT().NewProcessHandler(m.Any(), entrypoint.log).
					Return(entrypoint.process, test.errNewProcessHandler).Times(1)
				if test.errNewProcessHandler != nil {
					return test.errNewProcessHandler
				}
				mocks.process.EXPECT().Start().Times(1)
				return nil
			}()
			entrypoint.handleStatusChange()

			e.Equal(test.state, entrypoint.state)

			if expectedError != nil {
				e.Contains(logBuf.String(), expectedError.Error())
			}
		})
	}

	killTestCases := [...]struct {
		name    string
		state   State
		errKill error
	}{
		{name: "When state is inactive, notReady, alive and kill returns no error, should kill a process and change process state to changing",
			state: State{inactive, notReady, alive}},
		{name: "When state is inactive, changed, alive and kill returns no error, should kill a process and change process state to changing",
			state: State{inactive, changed, alive}},
		{name: "When state is inactive, updated, alive and kill returns no error, should kill a process and change process state to changing",
			state: State{inactive, updated, alive}},
		{name: "When state is inactive, applied, alive and kill returns no error, should kill a process and change process state to changing",
			state: State{inactive, applied, alive}},
		{name: "When state is inactive, notReady, alive and kill returns an error, should kill a process, change process state to changing and log an error",
			state:   State{inactive, notReady, alive},
			errKill: errors.New("kill error")},
	}
	for _, test := range killTestCases {
		test := test
		e.runWithMockEntrypoint(test.name, func(entrypoint *Entrypoint, mocks *mocksControl, logBuf *bytes.Buffer) {
			entrypoint.state = test.state
			mocks.process.EXPECT().Kill().Return(test.errKill).Times(1)
			entrypoint.handleStatusChange()

			if test.errKill == nil {
				test.state.process = changing
			} else {
				e.Contains(logBuf.String(), "could not kill an entrypoint")
			}
			e.Equal(test.state, entrypoint.state)
		})
	}

	configUpdateTestCases := [...]struct {
		name  string
		state State
	}{
		{name: "When state is inactive, changed, dead, should update configuration, increment configUpdatesRunning and change configuration state to notReady",
			state: State{inactive, changed, dead}},
		{name: "When state is active, changed, dead, should update configuration, increment configUpdatesRunning and change configuration state to notReady",
			state: State{active, changed, dead}},
		{name: "When state is active, changed, alive, should update configuration, increment configUpdatesRunning and change configuration state to notReady",
			state: State{active, changed, alive}},
	}
	for _, test := range configUpdateTestCases {
		test := test
		e.runWithMockEntrypoint(test.name, func(entrypoint *Entrypoint, mocks *mocksControl, _ *bytes.Buffer) {
			entrypoint.configUpdatesRunning = 0
			entrypoint.state = test.state
			mocks.configuration.EXPECT().Update().Times(1)
			entrypoint.handleStatusChange()

			e.Equal(1, entrypoint.configUpdatesRunning)
			test.state.configuration = notReady
			e.Equal(test.state, entrypoint.state)
		})
	}

	nothingToDoTestCases := [...]struct {
		state State
	}{
		{state: State{inactive, notReady, dead}},
		{state: State{inactive, notReady, changing}},
		{state: State{inactive, changed, changing}},
		{state: State{inactive, updated, dead}},
		{state: State{inactive, updated, changing}},
		{state: State{inactive, applied, dead}},
		{state: State{inactive, applied, changing}},
		{state: State{active, notReady, dead}},
		{state: State{active, notReady, changing}},
		{state: State{active, notReady, alive}},
		{state: State{active, changed, changing}},
		{state: State{active, updated, changing}},
		{state: State{active, applied, changing}},
		{state: State{active, applied, alive}},
	}
	for _, test := range nothingToDoTestCases {
		test := test
		e.runWithMockEntrypoint("When state is "+test.state.string()+", should do nothing", func(entrypoint *Entrypoint, mocks *mocksControl, _ *bytes.Buffer) {
			entrypoint.state = test.state
			entrypoint.handleStatusChange()

			e.Equal(test.state, entrypoint.state)
		})
	}
}
