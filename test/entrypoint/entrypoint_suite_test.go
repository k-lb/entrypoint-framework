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
	"log/slog"
	"testing"

	"test/internal/mocks"

	"github.com/k-lb/entrypoint-framework/handlers"
	"github.com/stretchr/testify/suite"
	m "go.uber.org/mock/gomock"
)

type EntrypointTestSuite struct {
	suite.Suite
}

type mocksControl struct {
	*m.Controller
	hc            *mocks.MockHandlersConstructorIface
	activation    *mocks.MockActivationHandler
	configuration *mocks.MockConfigurationHandler[handlers.UpdateResult]
	process       *mocks.MockProcessHandler
}

func (e *EntrypointTestSuite) runWithMockEntrypoint(
	name string, test func(*Entrypoint, *mocksControl, *bytes.Buffer)) {
	e.Run(name, func() {
		ctrl := m.NewController(e.T())
		defer ctrl.Finish()
		mocks := &mocksControl{
			Controller:    ctrl,
			hc:            mocks.NewMockHandlersConstructorIface(ctrl),
			activation:    mocks.NewMockActivationHandler(ctrl),
			configuration: mocks.NewMockConfigurationHandler[handlers.UpdateResult](ctrl),
			process:       mocks.NewMockProcessHandler(ctrl),
		}
		e.T().Parallel()
		logBuf := new(bytes.Buffer)
		test(
			&Entrypoint{
				log:           slog.New(slog.NewTextHandler(logBuf, nil)),
				hc:            mocks.hc,
				activation:    mocks.activation,
				configuration: mocks.configuration,
				process:       mocks.process,
			},
			mocks,
			logBuf,
		)
	})
}

func TestEntrypointTestSuite(t *testing.T) {
	suite.Run(t, new(EntrypointTestSuite))
}
