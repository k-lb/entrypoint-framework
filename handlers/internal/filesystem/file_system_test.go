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

package filesystem

import (
	"os"
	"path"
	"path/filepath"
)

func (f *filesystemTestSuite) TestDoesExist() {
	tests := [...]struct {
		name, fileName string
		exists         bool
	}{
		{name: "a file does not exist", fileName: "not_existing_file.test", exists: false},
		{name: "a directory does not exist", fileName: "not/existing/dir/file.test", exists: false},
		{name: "a file exists", fileName: "file.test", exists: true},
	}
	for _, test := range tests {
		f.RunWithTestDir("test DoesExist when "+test.name, func(testDir string) {
			testFile := path.Join(testDir, test.fileName)
			if test.exists {
				f.Require().NoError(os.WriteFile(testFile, []byte{}, 0664))
			}
			f.Equal(test.exists, f.DoesExist(testFile))
		})
	}
}

func (f *filesystemTestSuite) TestHardlink() {
	f.RunWithTestDir("when no files exist", func(testDir string) {
		err := f.Hardlink("not/existing/file", "not/existing/hardlink")
		f.Error(err, "should return an error")
	})

	f.RunWithTestDir("when a file exists", func(testDir string) {
		testFile := path.Join(testDir, "file.test")
		testFileContent := []byte("content")
		hardlinkFile := path.Join(testDir, "file.hardlink")
		f.Require().NoError(os.WriteFile(testFile, testFileContent, 0664))
		err := f.Hardlink(testFile, hardlinkFile)

		f.NoError(err)
		f.True(areFilesTheSame(testFile, hardlinkFile))

		newTestFile := path.Join(testDir, "new_file.test")
		f.Require().NoError(os.WriteFile(newTestFile, []byte("new content"), 0664))
		f.Require().NoError(os.Rename(newTestFile, testFile))

		hardlinkContent, err := os.ReadFile(hardlinkFile)
		f.Require().NoError(err)
		f.Equal(testFileContent, hardlinkContent)
	})

	f.RunWithTestDir("when a test file and a hardlink file already exists", func(testDir string) {
		testFile := path.Join(testDir, "file.test")
		hardlinkFile := path.Join(testDir, "file.hardlink")
		f.Require().NoError(os.WriteFile(testFile, []byte("content"), 0664))
		f.Require().NoError(f.Hardlink(testFile, hardlinkFile))
		f.Require().NoError(os.WriteFile(testFile, []byte("new content"), 0664))
		err := f.Hardlink(testFile, hardlinkFile)

		f.NoError(err)
		f.True(areFilesTheSame(testFile, hardlinkFile))
	})
}

func areFilesTheSame(filePath1, filePath2 string) bool {
	stat1, err1 := os.Stat(filePath1)
	stat2, err2 := os.Stat(filePath2)
	return err1 == nil && err2 == nil && os.SameFile(stat1, stat2)
}

func (f *filesystemTestSuite) TestDeleteFile() {
	f.Run("when a file does not exist", func() {
		f.NoError(f.DeleteFile("not_existing_file.test"))
	})

	f.RunWithTestDir("when a file exists", func(testDir string) {
		testFile := path.Join(testDir, "file.test")
		f.Require().NoError(os.WriteFile(testFile, []byte("content"), 0664))
		err := f.DeleteFile(testFile)

		f.NoError(err)
		f.False(f.DoesExist(testFile))
	})
}

func (f *filesystemTestSuite) TestClearDir() {
	f.RunWithTestDir("when a dir does not exist", func(testDir string) {
		f.NoError(f.ClearDir(path.Join(testDir, "not/existing/dir")))
	})

	f.RunWithTestDir("when a dir exists", func(testDir string) {
		testFiles := [...]string{
			path.Join(testDir, "file.test"),
			path.Join(testDir, "inner_dir", "file.test"),
			path.Join(testDir, "inner_dir", "inner_inner_dir", "file.test")}
		for _, testFile := range testFiles {
			f.Require().NoError(os.MkdirAll(path.Dir(testFile), os.ModePerm))
			f.Require().NoError(os.WriteFile(testFile, []byte("content"), 0664))
		}
		err := f.ClearDir(testDir)

		f.NoError(err)
		f.True(f.DoesExist(testDir))
		files, err := os.ReadDir(testDir)
		f.Require().NoError(err)
		f.Empty(files)
	})
}

func (f *filesystemTestSuite) TestCopyAndMoveFile() {
	presentFromFile := "fromFile.present"
	presentToFile := "toFile.present"
	absentFromFile := "fromFile.absent"
	absentToFile := "toFile.absent"
	copyMoveCases := [...]struct {
		name                     string
		copyOrMove               func(string, string) error
		sameFile, fromFileExists bool
	}{
		{name: "test move file ", copyOrMove: f.MoveFile, sameFile: true},
		{name: "test copy ", copyOrMove: f.Copy, fromFileExists: true},
	}
	for _, copyMoveTest := range copyMoveCases {
		copyMoveTest := copyMoveTest
		testCases := [...]struct {
			name, fromFile, fromContent, toContent, toFile string
			ignoreFromTestDir, ignoreToTestDir, expectErr  bool
		}{
			{name: "when both files do not exist", fromFile: absentFromFile, toFile: absentToFile, expectErr: true},
			{name: "when both files are empty strings", ignoreFromTestDir: true, ignoreToTestDir: true, expectErr: true},
			{name: `when only "from" file exists`, fromFile: presentFromFile, fromContent: "from content", toFile: absentToFile},
			{name: `when only "from" file exists and "to" file is an empty string`, fromFile: presentFromFile, fromContent: "from content", ignoreToTestDir: true, expectErr: true},
			{name: `when only "from" file exists with empty content`, fromFile: presentFromFile, toFile: absentToFile},
			{name: `when only "to" file exists`, fromFile: absentFromFile, toFile: presentToFile, expectErr: true},
			{name: `when only "to" file exists and "from" file is an empty string`, ignoreFromTestDir: true, toFile: presentToFile, expectErr: true},
			{name: "when both files exist", fromFile: presentFromFile, fromContent: "from content", toFile: presentToFile, toContent: "to content"},
			{name: "when both files exist and from file is empty", fromFile: presentFromFile, toFile: presentToFile, toContent: "to content"},
		}
		for _, test := range testCases {
			test := test
			f.RunWithTestDir(copyMoveTest.name+test.name, func(testDir string) {
				if !test.ignoreFromTestDir {
					test.fromFile = path.Join(testDir, test.fromFile)
				}
				if !test.ignoreToTestDir {
					test.toFile = path.Join(testDir, test.toFile)
				}
				f.Require().NoError(os.WriteFile(path.Join(testDir, presentFromFile), []byte(test.fromContent), 0664))
				fromStat, _ := os.Stat(test.fromFile)
				f.Require().NoError(os.WriteFile(path.Join(testDir, presentToFile), []byte(test.toContent), 0664))

				err := copyMoveTest.copyOrMove(test.fromFile, test.toFile)

				if test.expectErr {
					f.Error(err)
				} else {
					f.NoError(err)
					toStat, _ := os.Stat(test.toFile)
					f.Equal(copyMoveTest.sameFile, os.SameFile(fromStat, toStat))
					f.Equal(copyMoveTest.fromFileExists, f.DoesExist(test.fromFile))
				}
			})
		}
	}
}

func (f *filesystemTestSuite) TestListFileNamesInDir() {
	f.Run("when a directory does not exist", func() {
		files, err := f.ListFileNamesInDir("not/existing/dir")

		f.Error(err)
		f.Empty(files)
	})

	f.RunWithTestDir("when a directory exists with all kinds of files", func(testDir string) {
		expectedFiles := []string{"file1.test", "dir2/file2.test", "dir3/innter_dir/file3.test", "file1.hardlink", "file1.symlink"}
		for _, file := range expectedFiles[:3] {
			f.Require().NoError(os.MkdirAll(path.Join(testDir, filepath.Dir(file)), os.ModePerm))
			f.Require().NoError(os.WriteFile(path.Join(testDir, file), []byte{}, 0664))
		}
		f.Require().NoError(os.Link(path.Join(testDir, expectedFiles[0]), path.Join(testDir, expectedFiles[3])))
		f.Require().NoError(os.Symlink(path.Join(testDir, expectedFiles[0]), path.Join(testDir, expectedFiles[4])))

		files, err := f.ListFileNamesInDir(testDir)

		f.NoError(err)
		f.ElementsMatch(files, expectedFiles)
	})
	f.RunWithTestDir("when an empty directory exists", func(testDir string) {
		files, err := f.ListFileNamesInDir(testDir)
		f.NoError(err)
		f.Empty(files)
	})
}
