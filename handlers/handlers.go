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

// Package handlers provides three types of handlers.
//
// ActivationHandler provides information of a current state (active or inactive) of an application.
//
// ConfigurationHandler TODO
// ProcessHandler TODO
package handlers

import (
	"log/slog"

	"github.com/k-lb/entrypoint-framework/handlers/internal/filesystem"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"
)

const (
	handlerLogKey = "handler"
)

// ActivationHandler provides information of a current state (active or inactive) of application. It can be closed with
// Close method.
type ActivationHandler interface {
	GetWasChangedChannel() <-chan ActivationEvent
	Close()
}

// ActivationEvent contains a current state of an activation (active or inactive) and an error if it was observed.
type ActivationEvent struct {
	State bool
	Error error
}

// NewActivationHandler returns a new ActivationHandler and an error if any occurred. Activation is changed based on
// presence of an activationFile.
func NewActivationHandler(activationFile string, logger *slog.Logger) (*FileActivationHandler, error) {
	log := global.HandleNilLogger(logger).With(slog.String(handlerLogKey, "activation"), slog.String("file", activationFile))
	return newFileActivationHandler(activationFile, log, filesystem.New(log))
}
