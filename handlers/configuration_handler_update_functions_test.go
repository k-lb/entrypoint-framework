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
	"path"
)

func (h *HandlersTestSuite) TestUpdateSingleFileConfig() {
	h.RunWithMockEnv("when MoveFile returns an error, it returns an expected error", func(mocks *mocksControl) {
		errMoveFile := errors.New("move file error")
		mocks.fs.EXPECT().Copy("newConfigHardlinkPath", "oldConfigFile").Times(1).Return(errMoveFile)
		updateResult := updateSingleFileConfig("newConfigHardlinkPath", "oldConfigFile", mocks.fs)()

		h.Equal(errMoveFile, updateResult)
	})

	h.RunWithMockEnv("when MoveFile returns no error, it returns no error", func(mocks *mocksControl) {
		mocks.fs.EXPECT().Copy("newConfigHardlinkPath", "oldConfigFile").Times(1).Return(nil)
		updateResult := updateSingleFileConfig("newConfigHardlinkPath", "oldConfigFile", mocks.fs)()

		h.Nil(updateResult)
	})
}

func (h *HandlersTestSuite) TestUpdateTarredConfig() {
	type event struct {
		move, del  bool
		configFile string
		areDiff    bool
		err        error
	}
	testCases := [...]struct {
		name                                                  string
		errClearDir, errExtract, errListOldDir, errListNewDir error
		oldConfigFiles                                        []string
		newConfigFiles                                        []string
		events                                                []event
		expectedChangedFiles                                  map[string]Modification
	}{
		{name: "when ClearDir returns an error", errClearDir: errors.New("clear dir error")},
		{name: "when Extract returns an error", errExtract: errors.New("extract error")},
		{name: "when ListFileNamesInDir for oldConfigDir returns an error", errListOldDir: errors.New("list old dir error")},
		{name: "when ListFileNamesInDir for newConfigDir returns an error", errListNewDir: errors.New("list new dir error")},
		{name: "when ListFileNamesInDir returns empty maps", expectedChangedFiles: map[string]Modification{}},
		{name: "when MoveFile returns an error",
			newConfigFiles:       []string{"new"},
			events:               []event{{configFile: "new", move: true, err: errors.New("move file error")}},
			expectedChangedFiles: map[string]Modification{}},
		{name: "when DeleteFile returns an error",
			oldConfigFiles:       []string{"old"},
			events:               []event{{configFile: "old", del: true, err: errors.New("delete error")}},
			expectedChangedFiles: map[string]Modification{}},
		{name: "when AreFileContentsDifferent returns an error",
			newConfigFiles:       []string{"common"},
			oldConfigFiles:       []string{"common"},
			events:               []event{{configFile: "common", err: errors.New("are file contents different error")}},
			expectedChangedFiles: map[string]Modification{}},
		{name: "when AreFileContentsDifferent returns true, no error and MoveFile returns an error",
			newConfigFiles: []string{"common"},
			oldConfigFiles: []string{"common"},
			events: []event{{configFile: "common", areDiff: true},
				{configFile: "common", move: true, err: errors.New("move file error")}},
			expectedChangedFiles: map[string]Modification{}},
		{name: "when all type of file configuration is set and no errors occurred",
			newConfigFiles: []string{"new", "common the same", "common dif"},
			oldConfigFiles: []string{"common the same", "common dif", "old"},
			events: []event{{configFile: "new", move: true},
				{configFile: "common the same"},
				{configFile: "common dif", areDiff: true},
				{configFile: "common dif", move: true},
				{configFile: "old", del: true}},
			expectedChangedFiles: map[string]Modification{
				"new":        Created,
				"common dif": Modified,
				"old":        Deleted,
			}},
		{name: "when old dir is empty and no errors occurred",
			newConfigFiles: []string{"new", "other", "third"},
			oldConfigFiles: []string{},
			events: []event{{configFile: "new", move: true},
				{configFile: "other", move: true},
				{configFile: "third", move: true}},
			expectedChangedFiles: map[string]Modification{
				"new":   Created,
				"other": Created,
				"third": Created,
			}},
	}
	for _, test := range testCases {
		test := test
		h.RunWithMockEnv(test.name, func(mocks *mocksControl) {
			expectedError := func() error {
				if mocks.fs.EXPECT().ClearDir("newConfigDir").Times(1).Return(test.errClearDir); test.errClearDir != nil {
					return test.errClearDir
				}
				if mocks.fs.EXPECT().Extract("newConfigHardlinkPath", "newConfigDir").Times(1).Return(test.errExtract); test.errExtract != nil {
					return test.errExtract
				}
				if mocks.fs.EXPECT().ListFileNamesInDir("oldConfigDir").Times(1).Return(test.oldConfigFiles, test.errListOldDir); test.errListOldDir != nil {
					return test.errListOldDir
				}
				if mocks.fs.EXPECT().ListFileNamesInDir("newConfigDir").Times(1).Return(test.newConfigFiles, test.errListNewDir); test.errListNewDir != nil {
					return test.errListNewDir
				}
				for _, ev := range test.events {
					new, old := path.Join("newConfigDir", ev.configFile), path.Join("oldConfigDir", ev.configFile)
					if ev.move {
						mocks.fs.EXPECT().MoveFile(new, old).Times(1).Return(ev.err)
					} else if ev.del {
						mocks.fs.EXPECT().DeleteFile(old).Times(1).Return(ev.err)
					} else {
						mocks.fs.EXPECT().AreFilesDifferent(new, old).Times(1).Return(ev.areDiff, ev.err)
					}
					if ev.err != nil {
						return ev.err
					}
				}
				return nil
			}()

			updateResult := updateTarredConfig("newConfigHardlinkPath", "newConfigDir", "oldConfigDir", mocks.fs)()

			h.Equal(test.expectedChangedFiles, updateResult.ChangedFiles)
			h.ErrorIs(updateResult.Err, expectedError)
		})
	}
}

func (h *HandlersTestSuite) TestModificationToString() {
	h.Run("test Modification ToString", func() {
		h.Equal("deleted", Deleted.ToString())
		h.Equal("modified", Modified.ToString())
		h.Equal("created", Created.ToString())
		var m Modification
		h.Equal("invalid", m.ToString())
	})
}
