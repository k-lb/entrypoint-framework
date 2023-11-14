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
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"

	"github.com/k-lb/entrypoint-framework/handlers"
)

const (
	watchedActivationPath    = "/tmp/watched/activation/isactive"
	watchedConfigurationPath = "/tmp/watched/configuration/config.tar"
	newConfigurationDir      = "/tmp/configuration/new"
	oldConfigurationDir      = "/tmp/configuration/old"

	errKey = "error"
)

// Entrypoint contains all necessary variables for entrypoint to work.
type Entrypoint struct {
	activation           handlers.ActivationHandler
	configuration        handlers.ConfigurationHandler[handlers.UpdateResult]
	process              handlers.ProcessHandler
	state                State
	wasConfigChanged     bool
	configUpdatesRunning int

	log *slog.Logger
	hc  HandlersConstructorIface
}

// cmd returns an entrypoint command.
func cmd() *exec.Cmd {
	return exec.Command("sleep", "1")
}

func main() { // place here only the code that can't be tested
	for _, dir := range [...]string{path.Dir(watchedActivationPath), path.Dir(watchedConfigurationPath), oldConfigurationDir} {
		if err := os.MkdirAll(dir, fs.ModePerm); err != nil {
			panic(fmt.Sprintf("couldn't create directory \"%s\". Reason: %v", dir, err))
		}
	}
	e := Entrypoint{
		log: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.Level(-10)})),
		hc:  HandlersConstructor{},
	}

	err := e.initialize()
	defer e.tearDown()
	if err != nil {
		panic(fmt.Sprintf("couldn't initialize entrypoint. Reason: %v", err))
	}
	for {
		e.changeStateByEvent()
		e.log.Info("state was changed by an event", "state", e.state.string())
		e.handleStatusChange()
		e.log.Info("status change was handled    ", "state", e.state.string())
	}
}

// initialize prepares an Entrypoint instance by setting initial values and creating proper handlers.
func (e *Entrypoint) initialize() error {
	var err error
	e.wasConfigChanged = false
	e.configUpdatesRunning = 0
	e.activation, err = e.hc.NewActivationHandler(watchedActivationPath, e.log)
	if err != nil {
		return fmt.Errorf("could not create a new activation handler. Reason: %w", err)
	}
	e.configuration, err = e.hc.NewConfigurationHandler(
		watchedConfigurationPath, newConfigurationDir, oldConfigurationDir, e.log)
	if err != nil {
		return fmt.Errorf("could not create a new configuration handler. Reason: %w", err)
	}
	e.process, err = e.hc.NewProcessHandler(cmd(), e.log)
	if err != nil {
		return fmt.Errorf("could not create a new process handler. Reason: %w", err)
	}
	e.state = State{inactive, notReady, dead}
	return nil
}

// tearDown shutdowns all handlers making Entrypoint instance unusable.
func (e *Entrypoint) tearDown() {
	e.log.Info("tearing down entrypoint")
	e.activation.Close()
	e.configuration.Close()
	if err := e.process.Kill(); err != nil {
		e.log.Error("could not kill a process", slog.Any(errKey, err))
	}
}

// changeStateByEvent reacts on handlers events by changing state of the entrypoint.
func (e *Entrypoint) changeStateByEvent() {
	select {
	case ev := <-e.activation.GetWasChangedChannel():
		runFunctionIfNoError(e, ev, "activation was changed", e.activationWasChanged, ev.Error)
	case ev := <-e.configuration.GetWasChangedChannel():
		runFunctionIfNoError(e, ev, "configuration was changed", e.configurationWasChanged, ev)
	case ev := <-e.configuration.GetUpdateResultChannel():
		runFunctionIfNoError(e, ev, "configuration was updated", e.configurationWasUpdated, ev.Err)
	case ev := <-e.process.GetStartedChannel():
		runFunctionIfNoError(e, ev, "process was started", e.processWasStarted, ev)
	case ev := <-e.process.GetEndedChannel():
		e.processWasEnded(ev)
	}
}

// runFunctionIfNoError logs and runs f with ev argument only if err is nil.
func runFunctionIfNoError[T any](e *Entrypoint, ev T, eventDescription string, f func(T), err error) {
	e.log.Info(fmt.Sprintf("received %s event", eventDescription), slog.Any(errKey, err))
	if err == nil {
		f(ev)
	}
}

// activationWasChanged reacts to ActivationHandlers wasChanged event to change the entrypoint state.
func (e *Entrypoint) activationWasChanged(ev handlers.ActivationEvent) {
	e.state.activation = ActivationState(ev.State)
}

// configurationWasChanged reacts to ConfigurationHandlers wasChanged event to change the entrypoint state.
func (e *Entrypoint) configurationWasChanged(_ error) { e.state.configuration = changed }

// configurationWasUpdated reacts to event with configuration update results to change the entrypoint state.
func (e *Entrypoint) configurationWasUpdated(ev handlers.UpdateResult) {
	e.configUpdatesRunning--
	for file, modification := range ev.ChangedFiles {
		e.log.Info(fmt.Sprintf("File %s was %s", file, modification.ToString()))
	}
	if len(ev.ChangedFiles) > 0 {
		e.wasConfigChanged = true
	}
	if e.configUpdatesRunning == 0 {
		if e.wasConfigChanged {
			e.state.configuration = updated
		} else {
			e.state.configuration = applied
		}
	}
}

// processWasStarted reacts to event of starting process to change the entrypoint state.
func (e *Entrypoint) processWasStarted(_ error) {
	e.state.process = alive
	if e.state.configuration == updated {
		e.state.configuration = applied
		e.wasConfigChanged = false
	}
}

// processWasEnded reacts to event of stopping the process to change the entrypoint state.
func (e *Entrypoint) processWasEnded(ev error) {
	e.log.Info("received process was ended event", slog.Any(errKey, ev))
	e.state.process = dead
}

// handleStatusChange handles a status change.
func (e *Entrypoint) handleStatusChange() {
	if is(e.state).act(active).config(applied, updated).proc(dead).value() {
		e.start()
	} else if is(e.state).act(active).config(updated).proc(alive).value() {
		e.kill()
		if e.state.process == changing { //kill was successful
			e.start()
		}
	} else if is(e.state).act(inactive).proc(alive).value() {
		e.kill()
	} else if is(e.state).config(changed).proc(dead, alive).value() {
		e.configuration.Update()
		e.configUpdatesRunning++
		e.state.configuration = notReady
	}
}

// start creates a new process handler. If no errors occurred it starts the process and changes Entrypoints process
// state to changing.
func (e *Entrypoint) start() {
	var err error
	if e.process, err = e.hc.NewProcessHandler(cmd(), e.log); err != nil {
		e.log.Error("could not start an entrypoint", slog.Any(errKey, err))
		return
	}
	e.process.Start()
	e.state.process = changing
}

// kill kills an entrypoint's process. If no errors occurred it changes process state to changing.
func (e *Entrypoint) kill() {
	if err := e.process.Kill(); err != nil {
		e.log.Error("could not kill an entrypoint", slog.Any(errKey, err))
		return
	}
	e.state.process = changing
}
