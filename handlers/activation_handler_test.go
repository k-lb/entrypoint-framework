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
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/k-lb/entrypoint-framework/handlers/internal/filesystem"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"
)

func (h *HandlersTestSuite) TestFileActivationHandler() {
	const activationFile = "path/to/a/file.test"
	h.RunWithMockEnv("when a NewFileWatcher returns an error", func(mock *mocksControl) {
		mock.fs.EXPECT().NewFileWatcher(activationFile, fsnotify.Create|fsnotify.Remove).Times(1).Return(nil, errors.New("Watcher error"))
		handler, err := newFileActivationHandler(activationFile, logDiscard, mock.fs)

		h.Error(err)
		h.Nil(handler)
	})

	h.RunWithMockEnv("channel capacity and Close method", func(mock *mocksControl) {
		mock.init(activationFile, false)
		mock.watcher.EXPECT().Stop().Times(1)

		handler, err := newFileActivationHandler(activationFile, logDiscard, mock.fs)
		h.Require().NoError(err)
		h.Require().NotNil(handler)

		h.Equal(global.DefaultChanBuffSize, cap(handler.GetWasChangedChannel()))
		h.Zero(<-handler.GetWasChangedChannel())
		handler.Close()
		_, open := <-handler.done
		h.False(open)
		h.Nil(handler.GetWasChangedChannel())
		select {
		case <-handler.wasChanged:
			h.Fail("a wasChanged channel of an activation handler should not be closed after calling a Close method of the activation handler.")
		case <-time.After(time.Second / 100):
		}
	})

	watcherError := errors.New("watcher error")
	type Events = []struct {
		FileExists   bool
		WatcherError error
	}
	testCases := [...]struct {
		name              string
		initialFileExists bool
		events            Events
	}{
		{name: "a file is present without changes", initialFileExists: true},
		{name: "a file is absent without changes", initialFileExists: false},
		{name: "a file is present while initializing and deleted", initialFileExists: true, events: Events{{false, nil}}},
		{name: "a file is absent while initializing and created", initialFileExists: false, events: Events{{true, nil}}},
		{name: "a file is present while initializing, deleted and created", initialFileExists: true, events: Events{{false, nil}, {true, nil}}},
		{name: "a file is absent while initializing, created and deleted", initialFileExists: false, events: Events{{true, nil}, {false, nil}}},
		{name: "a file is present while initializing and the watcher returns an error", initialFileExists: true, events: Events{{true, watcherError}}},
		{name: "a file is absent while initializing and the watcher returns an error", initialFileExists: false, events: Events{{false, watcherError}}},
		{name: "a file is present while initializing, the watcher returns an error and the file is deleted", initialFileExists: true, events: Events{{true, watcherError}, {false, nil}}},
		{name: "a file is absent while initializing, the watcher returns an error and the file is created", initialFileExists: false, events: Events{{false, watcherError}, {true, nil}}},
	}
	for _, test := range testCases {
		h.RunWithMockEnv("when a watcher is not nil and "+test.name, func(mock *mocksControl) {
			filePresenceChanged := mock.init(activationFile, test.initialFileExists)
			handler, err := newFileActivationHandler(activationFile, logDiscard, mock.fs)

			for _, event := range test.events {
				mock.fs.EXPECT().DoesExist(activationFile).Times(1).Return(event.FileExists)
				mock.watcher.EXPECT().GetEvent().Times(1).Return(&filesystem.WatcherEvent{Error: event.WatcherError})
				filePresenceChanged <- struct{}{}
			}

			h.Require().NoError(err)
			expectedEvent := ActivationEvent{State: test.initialFileExists}
			h.Equal(expectedEvent, <-handler.GetWasChangedChannel(), "should push initial ActivationEvent to a channel")
			for _, testEvent := range test.events {
				expectedEvent := ActivationEvent{State: testEvent.FileExists, Error: testEvent.WatcherError}
				h.Equal(expectedEvent, <-handler.GetWasChangedChannel(), "should push expected ActivationEvent to a channel")
			}
			close(filePresenceChanged)
			_, open := <-handler.GetWasChangedChannel()
			h.False(open, "should close a channel")
		})
	}
	h.RunWithMockEnv("when a watcher is not nil and event is invalid", func(mock *mocksControl) {
		filePresenceChanged := mock.init(activationFile, false)
		handler, err := newFileActivationHandler(activationFile, logDiscard, mock.fs)
		h.Require().NotNil(handler)
		h.Require().NoError(err)
		mock.watcher.EXPECT().GetEvent().Times(1).Return(nil)
		<-handler.GetWasChangedChannel() // discard initial state
		filePresenceChanged <- struct{}{}
		close(filePresenceChanged)
		_, open := <-handler.GetWasChangedChannel()
		h.False(open, "should close a channel")
	})
}
