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

// Package filesystem implements functions that use file system operations.
// It is created to test functionality on a real operating system and to mock its usage in handlers package.
package filesystem

import (
	"log/slog"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"
)

// Interface Filesystem provides methods:
// - DoesExist which returns true if a file from path exists and false if it does not
// - NewFileWatcher which creates file watcher based on fsnotify library (inotify).
//
//go:generate mockgen -package=mocks -destination=../mocks/file_system_mock.go -source=file_system.go -mock_names=Filesystem=MockFilesystem
type Filesystem interface {
	DoesExist(string) bool
	NewFileWatcher(string, fsnotify.Op) (Watcher, error)
}

// New returns a Filesystem implementation that works on underlying filesystem.
func New(logger *slog.Logger) Filesystem {
	return real{log: global.HandleNilLogger(logger)}
}

// real implements Filesystem interface with methods using os library.
type real struct {
	log *slog.Logger
}

// DoesExist returns true if a file from path exists and false if it does not or an error occurs.
func (real) DoesExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
