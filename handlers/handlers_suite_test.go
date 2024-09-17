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
	"testing"

	"github.com/k-lb/entrypoint-framework/handlers/internal/global"
	"github.com/k-lb/entrypoint-framework/handlers/internal/mocks"

	"github.com/stretchr/testify/suite"
	m "go.uber.org/mock/gomock"
)

var logDiscard = global.HandleNilLogger(nil)

type HandlersTestSuite struct {
	suite.Suite
}

type mocksControl struct {
	*m.Controller
	fs      *mocks.MockFilesystem
	watcher *mocks.MockWatcher
}

func (mock *mocksControl) init(activationFile string, initialExists bool) chan struct{} {
	filePresenceChanged := make(chan struct{}, 1)
	mock.watcher.EXPECT().GetNotificationChannel().Times(1).Return(filePresenceChanged)
	mock.fs.EXPECT().NewFileWatcher(m.Any(), m.Any()).Times(1).Return(mock.watcher, nil)
	mock.fs.EXPECT().DoesExist(activationFile).Times(1).Return(initialExists)
	return filePresenceChanged
}

func (h *HandlersTestSuite) RunWithMockEnv(name string, test func(mocks *mocksControl)) {
	h.Run(name, func() {
		ctrl := m.NewController(h.T())
		defer ctrl.Finish()
		mc := &mocksControl{
			Controller: ctrl,
			fs:         mocks.NewMockFilesystem(ctrl),
			watcher:    mocks.NewMockWatcher(ctrl),
		}
		h.T().Parallel()
		test(mc)
	})
}

func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}
