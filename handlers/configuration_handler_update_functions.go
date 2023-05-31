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
	"path"

	"github.com/k-lb/entrypoint-framework/handlers/internal/filesystem"
)

// updateSingleFileConfig returns a function that copies a file from newConfigHardlinkPath to oldConfigFile.
func updateSingleFileConfig(newConfigHardlinkPath, oldConfigFile string, fs filesystem.Filesystem) func() error {
	return func() error {
		return fs.Copy(newConfigHardlinkPath, oldConfigFile)
	}
}

// updateTarredConfig returns a function that untars newConfigHardlinkPath into newConfigDir. Then it updates
// oldConfigDir to resemble newConfigDir. If a file hasn't changed it is not moved. It returns an UpdateResult.
func updateTarredConfig(newConfigHardlinkPath, newConfigDir, oldConfigDir string, fs filesystem.Filesystem) func() UpdateResult {
	return func() UpdateResult {
		if err := fs.ClearDir(newConfigDir); err != nil {
			return UpdateResult{Err: fmt.Errorf("could not clear a new config directory %s. Reason: %w", newConfigDir, err)}
		} else if err := fs.Extract(newConfigHardlinkPath, newConfigDir); err != nil {
			return UpdateResult{Err: fmt.Errorf("could not extract a file %s to a directory %s. Reason: %w", newConfigHardlinkPath, newConfigDir, err)}
		}
		filePresenceMap, err := createFilePresenceMap(oldConfigDir, newConfigDir, fs)
		if err != nil {
			return UpdateResult{Err: err}
		}
		changedFiles := map[string]Modification{}
		for configFile, flag := range filePresenceMap {
			newConfigFilePath := path.Join(newConfigDir, configFile)
			oldConfigFilePath := path.Join(oldConfigDir, configFile)
			switch flag {
			case newConfigDirFlag:
				if err := fs.MoveFile(newConfigFilePath, oldConfigFilePath); err != nil {
					return UpdateResult{changedFiles, fmt.Errorf("could not move a file. Result %w", err)}
				}
				changedFiles[configFile] = Created
			case newConfigDirFlag | oldConfigDirFlag:
				different, err := fs.AreFilesDifferent(newConfigFilePath, oldConfigFilePath)
				if err != nil {
					return UpdateResult{changedFiles, fmt.Errorf("could not check if files are different. Result %w", err)}
				}
				if different {
					if err := fs.MoveFile(newConfigFilePath, oldConfigFilePath); err != nil {
						return UpdateResult{changedFiles, fmt.Errorf("could not move a file . Result %w", err)}
					}
					changedFiles[configFile] = Modified
				}
			case oldConfigDirFlag:
				if err := fs.DeleteFile(oldConfigFilePath); err != nil {
					return UpdateResult{changedFiles, fmt.Errorf("could not delete a file. Result %w", err)}
				}
				changedFiles[configFile] = Deleted
			}
		}
		return UpdateResult{changedFiles, nil}
	}
}

// UpdateResult contains a map of file names with modification that was made to them and an error if it was observed.
type UpdateResult struct {
	ChangedFiles map[string]Modification
	Err          error
}

// Modification specifies type of modification made to a file while updating.
type Modification int

const (
	Deleted Modification = iota + 1
	Modified
	Created
)

// createFilePresenceMap creates a map of file's names from both oldConfigDir and newConfigDir with a presence in
// old/new ConfigDir flag.
func createFilePresenceMap(oldConfigDir, newConfigDir string, fs filesystem.Filesystem) (filePresenceMap, error) {
	result := filePresenceMap{}
	if err := result.setFlag(oldConfigDir, oldConfigDirFlag, fs); err != nil {
		return filePresenceMap{}, err
	} else if err := result.setFlag(newConfigDir, newConfigDirFlag, fs); err != nil {
		return filePresenceMap{}, err
	}
	return result, nil
}

type filePresenceMap map[string]int

const (
	oldConfigDirFlag int = 1 << iota
	newConfigDirFlag
)

// setFlag sets flag for each file from configDir into filePresenceMap.
func (f *filePresenceMap) setFlag(dir string, flag int, fs filesystem.Filesystem) error {
	configFiles, err := fs.ListFileNamesInDir(dir)
	if err != nil {
		return fmt.Errorf("could not list files in a dir: %s. Result %w", dir, err)
	}
	for _, configFile := range configFiles {
		(*f)[configFile] |= flag
	}
	return nil
}

// ToString returns string name of modification.
func (m Modification) ToString() string {
	switch m {
	case Deleted:
		return "deleted"
	case Modified:
		return "modified"
	case Created:
		return "created"
	}
	return "invalid"
}
