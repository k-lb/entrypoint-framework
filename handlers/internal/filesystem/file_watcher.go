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

//go:generate mockgen -package=mocks -destination=../mocks/file_watcher_mock.go -source=file_watcher.go -mock_names=Watcher=MockWatcher

package filesystem

import (
	"context"
	"fmt"
	"log/slog"
	"path"

	"github.com/fsnotify/fsnotify"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"
)

// Watcher describes types that are source of information about events (e.g. file change, errors, etc.).
// Watcher also notifies about such event via a channel.
type Watcher interface {
	// GetEvent returns file event that occurred in the system.
	GetEvent() *WatcherEvent
	// GetNotificationChannel returns a channel that sends notifications when a new event is available.
	GetNotificationChannel() <-chan struct{}
	// Stop causes Watcher to cease its operation.
	Stop()
}

// WatcherEvent is an event that a watcher pushes to a channel. It contains operation that was observed on a watched
// file or an error if it occurred. If the Error is not nil, Operation won't be set.
type WatcherEvent struct {
	// Operation denotes which action (e.g. write, read) was observed on the watched file.
	Operation fsnotify.Op
	// Error denotes that error has occurred while watching.
	Error error
}

// FileWatcher observes file and notifies when observed type of change occurs (e.g. write). It always provides latest
// event that has occurred.
type FileWatcher struct {
	notifier        *global.EventNotifier[WatcherEvent]
	fsnotifyWatcher *fsnotify.Watcher
}

// NewFileWatcher returns a watcher events channel and an error if any occurred. It initializes fsnotify watcher to a
// watchedFile and listens for its events in a new goroutine. A watcher event is pushed with an operation or an error
// depending on operation of fsnotify watcher. Watched operations can be created with "|" operator for example
// fsnotify.Create|fsnotify.Remove.
func (r real) NewFileWatcher(watchedFile string, watchedOps fsnotify.Op) (Watcher, error) {
	fsnotifyWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create a new fsnotify watcher. Reason: %w", err)
	}
	err = fsnotifyWatcher.Add(path.Dir(watchedFile))
	if err != nil {
		return nil, fmt.Errorf("could not add to fsnotify watcher a file: %s. Reason: %w", watchedFile, err)
	}
	fw := &FileWatcher{
		notifier:        global.NewEventNotifier[WatcherEvent](),
		fsnotifyWatcher: fsnotifyWatcher,
	}
	r.log.Debug("watching has started")

	go func() {
		defer fw.notifier.Stop()
		for {
			select {
			case ev, open := <-fw.fsnotifyWatcher.Events:
				if open {
					if ev.Op&watchedOps != 0 && ev.Name == watchedFile {
						fw.notifier.Notify(WatcherEvent{Operation: ev.Op})
						r.log.Debug("a watcher event was sent", slog.String("operation", ev.Op.String()))
					} else {
						r.log.Log(context.Background(), slog.LevelDebug-1, "an fsnotify event was observed", slog.String("event", ev.String()))
					}
				} else {
					r.log.Debug("a watcher events channel was closed")
					return
				}
			case err, open := <-fw.fsnotifyWatcher.Errors:
				if open {
					fw.notifier.Notify(WatcherEvent{Error: fmt.Errorf("watcher error. Reason: %w", err)})
					r.log.Debug("a watcher event was sent", slog.Any("error", err))
				} else {
					r.log.Debug("a watcher errors channel was closed")
					return
				}
			}
		}
	}()
	return fw, nil
}

// GetEvent returns the latest WatcherEvent that was observed. Nil will be returned if there were no new events
// between GetEvent calls.
func (f *FileWatcher) GetEvent() *WatcherEvent {
	return f.notifier.GetValue()
}

// GetNotificationChannel returns channel on which a notification that an event was observed is sent.
// To find out the latest event GetEvent must be called. There may be false positives. In such case GetEvent
// will return nil.
func (f *FileWatcher) GetNotificationChannel() <-chan struct{} {
	return f.notifier.GetNotifyChannel()
}

// Stop ceases FileWatcher operations.
func (f *FileWatcher) Stop() {
	f.fsnotifyWatcher.Close()
}
