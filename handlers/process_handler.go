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
	"fmt"
	"log/slog"
	"os/exec"
	"syscall"
)

// CmdProcessHandler executes an application and notifies when it starts and ends. It also allows to send signals to
// the process.
type CmdProcessHandler struct {
	cmd     *exec.Cmd
	started chan error
	ended   chan error
	log     *slog.Logger
}

// GetStartedChannel returns a read only channel with errors from a process start.
func (p *CmdProcessHandler) GetStartedChannel() <-chan error {
	return p.started
}

// GetEndedChannel returns a read only channel with errors from a process finish.
func (p *CmdProcessHandler) GetEndedChannel() <-chan error {
	return p.ended
}

// newCmdProcessHandler returns a pointer to a CmdProcessHandler and an error if any occurred.
func newCmdProcessHandler(cmd *exec.Cmd, log *slog.Logger) (*CmdProcessHandler, error) {
	if cmd == nil {
		return nil, errors.New("can not create process handler without a command")
	}
	if cmd.Err != nil {
		return nil, fmt.Errorf("process handler can not be initialized. Reason: %w", cmd.Err)
	}
	return &CmdProcessHandler{cmd: cmd, started: make(chan error, 1), ended: make(chan error, 1), log: log}, nil
}

// Start starts and waits for a command in a new goroutine. It returns start and wait errors to channels.
func (p *CmdProcessHandler) Start() {
	go func() {
		p.log.Info("starting a command")
		startErr := p.cmd.Start()
		p.started <- startErr
		p.log.Info("command start", slog.Any(errorKey, startErr))
		if startErr != nil {
			return
		}
		endErr := p.cmd.Wait()
		p.ended <- endErr
		p.log.Info("command end", slog.Any(errorKey, endErr))
	}()
}

// Stop sends sigterm signal to a process.
func (p *CmdProcessHandler) Stop() error { return p.Signal(syscall.SIGTERM) }

// Kill sends sigkill signal to a process.
func (p *CmdProcessHandler) Kill() error { return p.Signal(syscall.SIGKILL) }

// Signal sends a signal to a process if it's running and returns nil on success or an error.
func (p *CmdProcessHandler) Signal(signal syscall.Signal) error {
	if p.cmd.Process == nil {
		return fmt.Errorf("a process is nil. Can not send a signal %s", signal.String())
	}
	p.log.Info("a signal is being sent", slog.Any("signal", signal.String()))
	return p.cmd.Process.Signal(signal)
}
