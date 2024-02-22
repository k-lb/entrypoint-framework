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
	"log/slog"
	"os/exec"

	"github.com/k-lb/entrypoint-framework/handlers"
)

// HandlersConstructorIface interface is created for tests only.
//
//go:generate mockgen -package=mocks -destination=internal/mocks/handlers_constructor_mock.go -source=handlers_constructor.go -mock_names=HandlersConstructorIface=MockHandlersConstructor
//go:generate mockgen -package=mocks -destination=internal/mocks/handlers_mock.go -source=../../handlers/handlers.go
type HandlersConstructorIface interface {
	NewActivationHandler(activationFile string, logger *slog.Logger) (handlers.ActivationHandler, error)
	NewConfigurationHandler(newConfigFile, newConfigDir, oldConfigDir string, logger *slog.Logger) (handlers.ConfigurationHandler[handlers.UpdateResult], error)
	NewProcessHandler(cmd *exec.Cmd, logger *slog.Logger) (handlers.ProcessHandler, error)
}

// HandlersConstructor implements HandlersConstructorIface with calls to handlers package
type HandlersConstructor struct {
}

// NewActivationHandler returns a new ActivationHandler.
func (HandlersConstructor) NewActivationHandler(activationFile string, logger *slog.Logger) (handlers.ActivationHandler, error) {
	return handlers.NewActivationHandler(activationFile, logger)
}

// NewConfigurationHandler returns a new ConfigurationHandler.
func (HandlersConstructor) NewConfigurationHandler(newConfigFile, newConfigDir, oldConfigDir string, logger *slog.Logger) (handlers.ConfigurationHandler[handlers.UpdateResult], error) {
	return handlers.NewTarredConfigurationHandler(newConfigFile, newConfigDir, oldConfigDir, logger)
}

// NewProcessHandler returns a new ProcessHandler.
func (HandlersConstructor) NewProcessHandler(cmd *exec.Cmd, logger *slog.Logger) (handlers.ProcessHandler, error) {
	return handlers.NewProcessHandler(cmd, logger)
}
