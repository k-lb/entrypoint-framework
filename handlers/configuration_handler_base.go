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

// ConfigurationHandlerBase listens to changes of a configuration file (which should only be moved to by writer and
// hardlinked by reader). This triggers creation of a hardlink and pushing 'was changed' event. Then update can be done
// without any risk of reading/writing the same file.
type ConfigurationHandlerBase[T any] struct {
	wasChanged   chan error
	updateStart  chan struct{}
	updateFunc   func() T
	updateResult chan T
	isOpen       bool

	newConfigPath         string //a path to a new configuration.
	newConfigHardlinkPath string //a path to a hardlink of a new configuration.

	log *slog.Logger
	fs  filesystem.Filesystem
}

// GetWasChangedChannel returns a read only channel with an error when the configuration was changed. The error is nil
// when the configuration was handled successfully. When the handler is closed it returns a nil channel.
func (c *ConfigurationHandlerBase[_]) GetWasChangedChannel() <-chan error {
	if c.isOpen {
		return c.wasChanged
	}
	return nil
}

// Update triggers the configuration update. When the handler is closed it only logs an error.
func (c *ConfigurationHandlerBase[_]) Update() {
	if c.isOpen {
		c.updateStart <- struct{}{}
	} else {
		c.log.Error("can't update the configuration after handler was closed")
	}
}

// GetUpdateResultChannel returns a read only channel with a T event when the configuration was updated. When the
// handler is closed it returns a nil channel.
func (c *ConfigurationHandlerBase[T]) GetUpdateResultChannel() <-chan T {
	if c.isOpen {
		return c.updateResult
	}
	return nil
}

// Close triggers closing of the ConfigurationHandlerBase.
func (c *ConfigurationHandlerBase[_]) Close() {
	if c.isOpen {
		close(c.updateStart)
		c.isOpen = false
	}
}

// newConfigurationHandlerBase returns a pointer to a ConfigurationHandlerBase and an error if any occurred. It
// initializes a file watcher, handles an initial configuration if present and listen for configuration changes in a new
// goroutine.
func newConfigurationHandlerBase[T any](
	newConfigPath,
	newConfigHardlinkPath string,
	updateFunc func() T,
	log *slog.Logger,
	fs filesystem.Filesystem) (*ConfigurationHandlerBase[T], error) {
	c := &ConfigurationHandlerBase[T]{
		wasChanged:   make(chan error, global.DefaultChanBuffSize),
		updateStart:  make(chan struct{}, global.DefaultChanBuffSize),
		updateResult: make(chan T, global.DefaultChanBuffSize),
		isOpen:       true,

		newConfigPath:         newConfigPath,
		newConfigHardlinkPath: newConfigHardlinkPath,
		updateFunc:            updateFunc,

		log: log,
		fs:  fs,
	}
	fw, err := fs.NewFileWatcher(newConfigPath, fsnotify.Create)
	if err != nil {
		return nil, fmt.Errorf("could not create a new file watcher for a file: %s. Reason: %w", newConfigPath, err)
	}

	if fs.DoesExist(newConfigPath) {
		c.handle(new(filesystem.WatcherEvent))
	}
	go c.listenToEvents(fw)
	return c, nil
}

// handle pushes a handling error to wasChanged channel and logs it.
func (c *ConfigurationHandlerBase[_]) handle(ev *filesystem.WatcherEvent) {
	if ev == nil { // ignore invalidated events
		return
	}
	err := ev.Error
	if err != nil {
		err = fmt.Errorf("error from watcher(%s). Reason: %w", c.newConfigPath, err)
	} else if err = c.fs.Hardlink(c.newConfigPath, c.newConfigHardlinkPath); err != nil {
		err = fmt.Errorf("could not create a hardlink of a file %s to %s. Reason: %w", c.newConfigPath, c.newConfigHardlinkPath, err)
	}
	c.wasChanged <- err
	c.log.Debug("A wasChanged event was sent", slog.Any(errorKey, err))
}

// listenToEvents listens to changes of a new configuration from watcher and an update channel.
func (c *ConfigurationHandlerBase[_]) listenToEvents(fw filesystem.Watcher) {
	configChanged := fw.GetNotificationChannel()
	for {
		select {
		case _, open := <-configChanged:
			if open {
				c.handle(fw.GetEvent())
			} else {
				configChanged = nil
				if err := c.fs.DeleteFile(c.newConfigHardlinkPath); err != nil {
					c.wasChanged <- err
				}
				close(c.wasChanged)
				c.log.Debug("A wasChanged channel was closed")
			}
		case _, open := <-c.updateStart:
			if !open {
				fw.Stop()
				c.updateStart = nil
				close(c.updateResult)
				c.log.Debug("An update result channel was closed")
				continue
			}
			if c.updateFunc != nil {
				c.updateResult <- c.updateFunc()
				c.log.Debug("An update result event was sent")
			}
		}
		if configChanged == nil && c.updateStart == nil {
			return
		}
	}
}
