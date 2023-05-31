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

package handlers

import (
	"errors"

	"github.com/fsnotify/fsnotify"
	"github.com/k-lb/entrypoint-framework/handlers/internal/filesystem"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"
)

func (h *HandlersTestSuite) TestNewConfigurationHandlerBase() {
	errHardlink := errors.New("hardlink error")
	errFsnotify := errors.New("fsnotify error")
	neverUsedUpdateFunc := func() int { h.Fail("updateFunc called"); return 0 }

	h.RunWithMockEnv("when NewFileWatcher returns an error, should returns a nil handler and an error", func(mocks *mocksControl) {
		watcherErr := errors.New("watcher error")
		mocks.fs.EXPECT().NewFileWatcher("newConfigPath", fsnotify.Create).Times(1).Return(mocks.watcher, watcherErr)
		configHandler, err := newConfigurationHandlerBase("newConfigPath", "newConfigHardlinkPath", neverUsedUpdateFunc, logDiscard, mocks.fs)

		h.Nil(configHandler)
		h.Error(err)
		h.ErrorIs(err, watcherErr)
	})

	h.runWithExpects("when NewFileWatcher returns no error, should set all configuration handler's fields and correctly close them", func(configChanged chan struct{}, mocks *mocksControl) *ConfigurationHandlerBase[int] {
		expectedUpdateResult := 123
		mocks.fs.EXPECT().DoesExist("newConfigPath").Times(1).Return(false)
		configHandler, err := newConfigurationHandlerBase("newConfigPath", "newConfigHardlinkPath", func() int { return expectedUpdateResult }, logDiscard, mocks.fs)

		h.NoError(err)
		h.NotNil(configHandler)
		h.Equal(global.DefaultChanBuffSize, cap(configHandler.GetWasChangedChannel()))
		h.Equal(global.DefaultChanBuffSize, cap(configHandler.updateStart))
		h.Equal(global.DefaultChanBuffSize, cap(configHandler.GetUpdateResultChannel()))
		h.True(configHandler.isOpen)
		h.Equal("newConfigPath", configHandler.newConfigPath)
		h.Equal("newConfigHardlinkPath", configHandler.newConfigHardlinkPath)
		h.Equal(expectedUpdateResult, configHandler.updateFunc())

		return configHandler
	})

	initialConfigTestCases := [...]struct {
		name            string
		doesConfigExist bool
		hardlinkError   error
	}{
		{name: "when a new config file doesn't exist, shouldn't push an event", doesConfigExist: false},
		{name: "when a new config file exists and a hardlink function returns an error, should push event with the error", doesConfigExist: true, hardlinkError: errHardlink},
		{name: "when a new config file exists and a hardlink function doesn't return an error, should push event with a nil", doesConfigExist: true},
	}
	for _, test := range initialConfigTestCases {
		h.runWithExpects(test.name, func(configChanged chan struct{}, mocks *mocksControl) *ConfigurationHandlerBase[int] {
			mocks.fs.EXPECT().DoesExist("newConfigPath").Times(1).Return(test.doesConfigExist)
			if test.doesConfigExist {
				mocks.fs.EXPECT().Hardlink("newConfigPath", "newConfigHardlinkPath").Times(1).Return(test.hardlinkError)
			}
			configHandler, err := newConfigurationHandlerBase("newConfigPath", "newConfigHardlinkPath", neverUsedUpdateFunc, logDiscard, mocks.fs)

			h.NoError(err)
			h.NotNil(configHandler)
			if test.doesConfigExist {
				wasChangedError := <-configHandler.GetWasChangedChannel()
				if test.hardlinkError != nil {
					h.Error(wasChangedError)
					h.ErrorIs(err, test.hardlinkError)
				} else {
					h.NoError(wasChangedError)
				}
			}
			return configHandler
		})
	}

	configChangedTestCases := [...]struct {
		name   string
		events []any
	}{
		{name: "a fsnotify error event has occurred", events: []any{errFsnotify}},
		{name: "a hardlink error event has occurred", events: []any{errHardlink}},
		{name: "an event without errors has occurred", events: []any{nil}},
		{name: "events with no errors, a fsnotify error and a hardlink error have occurred", events: []any{nil, errFsnotify, errHardlink}},
		{name: "an update method was called without configuration changes", events: []any{1}},
		{name: "an update method was called after configuration was changed", events: []any{nil, 1}},
		{name: "an update method was called two times in a row", events: []any{1, 2}},
		{name: "an update method was called multiple times after configuration was changed", events: []any{nil, 1, nil, 2}},
	}
	for _, test := range configChangedTestCases {
		test := test
		h.runWithExpects("when a new config file doesn't exist and "+test.name, func(configChanged chan struct{}, mocks *mocksControl) *ConfigurationHandlerBase[int] {
			mocks.fs.EXPECT().DoesExist("newConfigPath").Times(1).Return(false)
			count := 0
			countUpdateFunc := func() int { count++; return count }
			configHandler, err := newConfigurationHandlerBase("newConfigPath", "newConfigHardlinkPath", countUpdateFunc, logDiscard, mocks.fs)
			h.Require().NotNil(configHandler)
			h.Require().NoError(err)

			for _, ev := range test.events {
				if errWasChanged, ok := ev.(error); ok || ev == nil {
					if errWasChanged == errFsnotify {
						mocks.watcher.EXPECT().GetEvent().Times(1).Return(&filesystem.WatcherEvent{Error: errFsnotify})
					} else {
						mocks.watcher.EXPECT().GetEvent().Times(1).Return(&filesystem.WatcherEvent{})
						mocks.fs.EXPECT().Hardlink("newConfigPath", "newConfigHardlinkPath").Times(1).Return(errWasChanged)
					}
					configChanged <- struct{}{}
					h.ErrorIs(<-configHandler.GetWasChangedChannel(), errWasChanged)
				} else if updateResult, ok := ev.(int); ok {
					configHandler.Update()
					if updateResult > 0 {
						h.Equal(<-configHandler.GetUpdateResultChannel(), updateResult)
					}
				} else {
					h.Fail("wrong event")
				}
			}

			return configHandler
		})
	}

	h.RunWithMockEnv("when Update is called after handler was closed", func(mocks *mocksControl) {
		configChanged := make(chan struct{}, 10)
		mocks.fs.EXPECT().NewFileWatcher("newConfigPath", fsnotify.Create).Times(1).Return(mocks.watcher, nil)
		mocks.fs.EXPECT().DoesExist("newConfigPath").Times(1).Return(false)
		errDeleteHardlink := errors.New("delete hardlink error")
		mocks.fs.EXPECT().DeleteFile("newConfigHardlinkPath").Times(1).Return(errDeleteHardlink)
		mocks.watcher.EXPECT().GetNotificationChannel().Times(1).Return(configChanged)
		mocks.watcher.EXPECT().Stop().Times(1)
		configHandler, err := newConfigurationHandlerBase("newConfigPath", "newConfigHardlinkPath", neverUsedUpdateFunc, logDiscard, mocks.fs)
		h.Require().NotNil(configHandler)
		h.Require().NoError(err)

		configHandler.Close()
		configHandler.Update()

		_, open := <-configHandler.updateResult
		h.False(open)
		close(configChanged)
		h.ErrorIs(<-configHandler.wasChanged, errDeleteHardlink)
		_, open = <-configHandler.wasChanged
		h.False(open)
	})

	h.runWithExpects("when an event is nil", func(configChanged chan struct{}, mocks *mocksControl) *ConfigurationHandlerBase[int] {
		mocks.fs.EXPECT().DoesExist("newConfigPath").Times(1).Return(false)
		configHandler, err := newConfigurationHandlerBase("newConfigPath", "newConfigHardlinkPath", neverUsedUpdateFunc, logDiscard, mocks.fs)
		h.Require().NotNil(configHandler)
		h.Require().NoError(err)
		mocks.watcher.EXPECT().GetEvent().Times(1).Return(nil)
		configChanged <- struct{}{}
		return configHandler
	})
}

func (h *HandlersTestSuite) runWithExpects(name string, test func(chan struct{}, *mocksControl) *ConfigurationHandlerBase[int]) {
	h.RunWithMockEnv(name, func(mocks *mocksControl) {
		mocks.fs.EXPECT().NewFileWatcher("newConfigPath", fsnotify.Create).Times(1).Return(mocks.watcher, nil)
		mocks.fs.EXPECT().DeleteFile("newConfigHardlinkPath").Times(1).Return(nil)
		configChanged := make(chan struct{}, 10)
		mocks.watcher.EXPECT().GetNotificationChannel().Times(1).Return(configChanged)

		configHandler := test(configChanged, mocks)

		close(configChanged)
		_, open := <-configHandler.wasChanged
		h.False(open)

		mocks.watcher.EXPECT().Stop().Times(1)
		configHandler.Close()
		_, open = <-configHandler.updateResult
		h.False(open, name)
		h.Nil(configHandler.updateStart)
		h.Nil(configHandler.GetWasChangedChannel())
		h.Nil(configHandler.GetUpdateResultChannel())
	})
}
