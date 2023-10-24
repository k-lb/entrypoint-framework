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
	"os/exec"
	"strings"
	"time"
)

func (h *HandlersTestSuite) TestCmdProcessHandler() {
	cmd := func(command string) *exec.Cmd {
		args := strings.Fields(command)
		if len(args) == 0 {
			return nil
		}
		return exec.Command(args[0], args[1:]...)
	}
	expectNoEvents := func(c <-chan error) {
		select {
		case <-c:
			h.Fail("event is not expected to be sent to this channel")
		case <-time.After(time.Second / 10):
		}
	}

	testCases := [...]struct {
		name                              string
		command                           string
		expectNoEndedEvent, expectedError bool
	}{
		{
			name:          "when an empty command is passed, it returns nil handler and an error",
			expectedError: true,
		}, {
			name:          "when a command is invalid, it returns nil handler and an error",
			command:       "not-existing-command",
			expectedError: true,
		}, {
			name:    "when starting a command returns no error and a process exits immediately, it returns a nil ended event",
			command: "echo",
		},
		{name: "when starting a command returns no error and a process exits with some delay, it returns no ended event",
			command:            "sleep 1",
			expectNoEndedEvent: true,
		},
	}
	for _, test := range testCases {
		test := test
		h.Run(test.name, func() {
			h.T().Parallel()
			handler, err := newCmdProcessHandler(cmd(test.command), logDiscard)

			if test.expectedError {
				h.Error(err)
				h.Nil(handler)
				return
			}

			h.NoError(err)
			h.NotNil(handler)
			h.Equal(1, cap(handler.started))
			h.Equal(1, cap(handler.ended))

			handler.Start()
			h.NoError(<-handler.GetStartedChannel())
			if test.expectNoEndedEvent {
				expectNoEvents(handler.GetEndedChannel())
				h.Require().NoError(handler.Stop())
				h.Error(<-handler.GetEndedChannel())
				return
			}

			h.Nil(<-handler.GetEndedChannel())
		})
	}

	h.Run("when Kill is called but process is nil, it returns an error", func() {
		h.T().Parallel()
		handler, err := newCmdProcessHandler(cmd("echo"), logDiscard)

		h.Require().NoError(err)
		h.Require().NotNil(handler)
		h.EqualError(handler.Kill(), "a process is nil. Can not send a signal killed")
	})
}
