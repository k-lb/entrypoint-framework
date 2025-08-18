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
	"sync/atomic"

	"github.com/k-lb/entrypoint-framework/handlers/internal/filesystem"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"

	"github.com/fsnotify/fsnotify"
)

// ConfigurationHandlerBase listens to changes of a configuration file (which should only be moved to by writer and
// hardlinked by reader). This triggers creation of a hardlink and pushing 'was changed' event. Then update can be done
// without any risk of reading/writing the same file.
type ConfigurationHandlerBase[T any] struct {
	wasChangedCh    chan error
	wasChanged      *atomic.Bool
	updateStartCh   chan struct{}
	isUpdateRunning *atomic.Bool
	updateFunc      func() T
	updateResultCh  chan T
	isOpen          bool

	newConfigPath         string //a path to a new configuration.
	newConfigHardlinkPath string //a path to a hardlink of a new configuration.

	log *slog.Logger
	fs  filesystem.Filesystem
}

// GetWasChangedChannel returns a read only channel with an error that occurred during configuration changing. The error
// is nil when the configuration was changed successfully. When the handler is closed it returns a nil channel.
func (c *ConfigurationHandlerBase[_]) GetWasChangedChannel() <-chan error {
	if c.isOpen {
		return c.wasChangedCh
	}
	return nil
}

// Update triggers the configuration update. When the handler is closed it only logs an error.
func (c *ConfigurationHandlerBase[_]) Update() error {
	if !c.isOpen {
		return errors.New("can't update the configuration after handler was closed")
	}
	if !c.wasChanged.Load() {
		return errors.New("an Update was called without configuration changes")
	}
	if c.isUpdateRunning.Load() {
		return errors.New("an Update was called before previous update of configuration was finished")
	}
	if len(c.updateResultCh) > 0 {
		return errors.New("an Update was called before previous configuration result was read")
	}
	c.updateStartCh <- struct{}{}
	c.isUpdateRunning.Store(true)
	return nil
}

// GetUpdateResultChannel returns a read only channel with a T event when the configuration was updated. When the
// handler is closed it returns a nil channel.
func (c *ConfigurationHandlerBase[T]) GetUpdateResultChannel() <-chan T {
	if c.isOpen {
		return c.updateResultCh
	}
	return nil
}

// Close triggers closing of the ConfigurationHandlerBase.
func (c *ConfigurationHandlerBase[_]) Close() {
	if c.isOpen {
		close(c.updateStartCh)
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
		wasChangedCh:    make(chan error, global.DefaultChanBuffSize),
		wasChanged:      &atomic.Bool{},
		updateStartCh:   make(chan struct{}, global.DefaultChanBuffSize),
		isUpdateRunning: &atomic.Bool{},
		updateResultCh:  make(chan T, global.DefaultChanBuffSize),
		isOpen:          true,

		newConfigPath:         newConfigPath,
		newConfigHardlinkPath: newConfigHardlinkPath,
		updateFunc:            updateFunc,

		log: log,
		fs:  fs,
	}
	c.wasChanged.Store(false)
	c.isUpdateRunning.Store(false)
	fw, err := fs.NewFileWatcher(newConfigPath, fsnotify.Create|fsnotify.Remove)
	if err != nil {
		return nil, fmt.Errorf("could not create a new file watcher for a file: %s. Reason: %w", newConfigPath, err)
	}

	if fs.DoesExist(newConfigPath) {
		c.handle(new(filesystem.WatcherEvent))
	}
	go c.listenToEvents(fw)
	return c, nil
}

var ErrConfigDeleted = errors.New("configuration was deleted")

// handle pushes a handling error to wasChanged channel and logs it.
func (c *ConfigurationHandlerBase[_]) handle(ev *filesystem.WatcherEvent) {
	if ev == nil { // ignore invalidated events
		return
	}
	err := ev.Error
	if err != nil {
		err = fmt.Errorf("error from watcher(%s). Reason: %w", c.newConfigPath, err)
	} else if ev.Operation.Has(fsnotify.Remove) {
		err = ErrConfigDeleted
	} else if err = c.fs.Hardlink(c.newConfigPath, c.newConfigHardlinkPath); err != nil {
		err = fmt.Errorf("could not create a hardlink of a file %s to %s. Reason: %w", c.newConfigPath, c.newConfigHardlinkPath, err)
	} else {
		c.wasChanged.Store(true)
	}
	c.wasChangedCh <- err
	c.log.Debug("A wasChanged event was sent", slog.Any(errorKey, err))
}

// listenToEvents listens to changes of a new configuration from watcher and an update channel.
func (c *ConfigurationHandlerBase[_]) listenToEvents(fw filesystem.Watcher) {
	configChangedCh := fw.GetNotificationChannel()
	for {
		select {
		case _, open := <-configChangedCh:
			if open {
				c.handle(fw.GetEvent())
				continue
			}
			configChangedCh = nil
			if err := c.fs.DeleteFile(c.newConfigHardlinkPath); err != nil {
				// c.log.Error("could not delete a file", slog.String("file", c.newConfigHardlinkPath), slog.Any("error", err))
				c.wasChangedCh <- err
			}
			close(c.wasChangedCh)
			c.log.Debug("A wasChanged channel was closed")

		case _, open := <-c.updateStartCh:
			if open && c.updateFunc != nil {
				c.updateResultCh <- c.updateFunc()
				c.log.Debug("An update result event was sent")
			} else if !open {
				c.updateStartCh = nil
				fw.Stop()
				close(c.updateResultCh)
				c.log.Debug("An update result channel was closed")
			}
			c.wasChanged.Store(false)
			c.isUpdateRunning.Store(false)
		}
		if configChangedCh == nil && c.updateStartCh == nil {
			return
		}
	}
}
