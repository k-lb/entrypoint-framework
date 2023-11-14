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
	"fmt"
	"log/slog"

	"github.com/k-lb/entrypoint-framework/handlers/internal/filesystem"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"

	"github.com/fsnotify/fsnotify"
)

// FileActivationHandler implements ActivationHandler interface. It uses provided file as a source for ActivationEvents.
type FileActivationHandler struct {
	wasChanged     chan ActivationEvent
	done           chan bool
	activationFile string
	log            *slog.Logger
	fs             filesystem.Filesystem

	isOpen bool
}

// GetWasChangedChannel returns a read only channel with an ActivationEvent when the activation was changed. When the
// handler is closed it returns a nil channel.
func (a *FileActivationHandler) GetWasChangedChannel() <-chan ActivationEvent {
	if a.isOpen {
		return a.wasChanged
	}
	return nil
}

// Close triggers closing of the FileActivationHandler.
func (a *FileActivationHandler) Close() {
	if a.isOpen {
		close(a.done)
		a.isOpen = false
	}
}

// newFileActivationHandler returns a pointer to a FileActivationHandler and an error if any occurred. It initializes a
// file watcher, handles an initial activation and listen for activation changes in a new goroutine.
func newFileActivationHandler(activationFile string, log *slog.Logger, fs filesystem.Filesystem) (*FileActivationHandler, error) {
	a := &FileActivationHandler{
		wasChanged:     make(chan ActivationEvent, global.DefaultChanBuffSize),
		done:           make(chan bool),
		activationFile: activationFile,
		log:            log,
		fs:             fs,
		isOpen:         true,
	}
	fw, err := fs.NewFileWatcher(activationFile, fsnotify.Create|fsnotify.Remove)
	if err != nil {
		return nil, fmt.Errorf("could not create a new file watcher for a file: %s. Reason: %w", activationFile, err)
	}

	a.handle(new(filesystem.WatcherEvent))
	go a.listenActivationChanges(fw)
	return a, nil
}

// handle pushes an ActivationEvent to wasChanged channel and logs it.
func (a *FileActivationHandler) handle(ev *filesystem.WatcherEvent) {
	if ev == nil { // ignore invalidated events
		return
	}
	event := ActivationEvent{State: a.fs.DoesExist(a.activationFile), Error: ev.Error}
	a.wasChanged <- event
	a.log.Debug("an event was sent", slog.Bool("state", event.State), slog.Any(errorKey, event.Error))
}

// listenActivationChanges listens to a filePresenceChanged channel and handle its events or closure.
func (a *FileActivationHandler) listenActivationChanges(fw filesystem.Watcher) {
	notifier := fw.GetNotificationChannel()
	for {
		select {
		case _, open := <-notifier:
			if open {
				a.handle(fw.GetEvent())
			} else {
				close(a.wasChanged)
				a.log.Debug("a wasChange channel was closed")
				return
			}
		case <-a.done:
			fw.Stop()
			return
		}
	}
}
