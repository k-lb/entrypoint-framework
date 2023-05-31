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
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/k-lb/entrypoint-framework/handlers/internal/global"
)

// Filesystem provides multiple file system utilities.
// It is also used to separate file operations from rest of the code.
//
//go:generate mockgen -package=mocks -destination=../mocks/file_system_mock.go -source=file_system.go -mock_names=Filesystem=MockFilesystem
type Filesystem interface {
	// DoesExist returns true if a status from path returns no error.
	DoesExist(path string) bool
	// Hardlink creates a hardlink of filePath to hardlinkPath. If hardlinkPath already exists then it is deleted.
	Hardlink(filePath, hardlinkPath string) error
	// DeleteFile deletes a filePath.
	DeleteFile(filePath string) error
	// ClearDir deletes all files from a dirPath.
	ClearDir(filePath string) error
	// MoveFile moves a fromPath file to a toPath.
	MoveFile(fromPath, toPath string) error
	// Copy copies a fromPath file content to a toPath file.
	Copy(fromPath, toPath string) error
	// ListFileNamesInDir returns a list with file names (not paths) from dirPath.
	ListFileNamesInDir(dirPath string) ([]string, error)
	// NewFileWatcher creates file watcher based on fsnotify library (inotify).
	NewFileWatcher(watchedFile string, watchedOps fsnotify.Op) (Watcher, error)
	// Extract extracts all files from a tarball to a toDir directory.
	Extract(tarball, toDir string) error
	// AreFilesDifferent checks if two files has different contents or modes.
	AreFilesDifferent(firstFilePath, secondFilePath string) (bool, error)
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

// Hardlink creates a hardlink of filePath to hardlinkPath. If hardlinkPath already exists then it is deleted.
func (r real) Hardlink(filePath, hardlinkPath string) error {
	if err := r.DeleteFile(hardlinkPath); err != nil {
		return err
	}
	return os.Link(filePath, hardlinkPath)
}

// DeleteFile deletes a filePath.
func (r real) DeleteFile(filePath string) error {
	if !r.DoesExist(filePath) {
		return nil
	}
	return os.Remove(filePath)
}

// ClearDir deletes all files from a dirPath.
func (real) ClearDir(dirPath string) error {
	if err := os.RemoveAll(dirPath); err != nil {
		return err
	}
	return os.MkdirAll(dirPath, os.ModePerm)
}

// MoveFile moves a fromPath file to a toPath.
func (real) MoveFile(fromPath, toPath string) error {
	return os.Rename(fromPath, toPath)
}

// Copy copies a fromPath file content to a toPath file.
func (real) Copy(fromPath, toPath string) error {
	content, err := os.ReadFile(fromPath)
	if err != nil {
		return err
	}
	return os.WriteFile(toPath, content, os.ModePerm)
}

// ListFileNamesInDir returns a list with file names (not paths) from dirPath.
func (real) ListFileNamesInDir(dirPath string) ([]string, error) {
	return listFileNamesInDir(dirPath, "")
}

func listFileNamesInDir(dirPath, dirName string) ([]string, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	fileNameList := []string{}
	for _, dirEntry := range dirEntries {
		path := filepath.Join(dirPath, dirEntry.Name())
		fileName := filepath.Join(dirName, dirEntry.Name())
		if dirEntry.Type().IsDir() {
			if innerFileNameList, err := listFileNamesInDir(path, fileName); err != nil {
				return nil, err
			} else {
				fileNameList = append(fileNameList, innerFileNameList...)
			}
			continue
		}
		if !(dirEntry.Type().IsRegular() || dirEntry.Type()&fs.ModeSymlink != 0) {
			return nil, fmt.Errorf("%s is not a regular file or symlink, type: %s", filepath.Join(dirPath, dirEntry.Name()), dirEntry.Type().String())
		}
		fileNameList = append(fileNameList, fileName)
	}
	return fileNameList, nil
}
