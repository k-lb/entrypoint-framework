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
	"os/exec"
	"path"
)

func (f *filesystemTestSuite) TestExtract() {
	f.Run("when a file does not exist", func() {
		err := f.Extract("not/existing/file.test", "not/existing/dir")

		f.Error(err)
	})

	f.RunWithTestDir("when there are multiple files", func(testDir string) {
		files := []string{"file.test", "dir", "dir/inner_file.test", "file1.symlink", "file1.hardlink"}
		f.Require().NoError(os.WriteFile(path.Join(testDir, files[0]), []byte("file content"), 0664))
		f.Require().NoError(os.Mkdir(path.Join(testDir, files[1]), os.ModePerm))
		f.Require().NoError(os.WriteFile(path.Join(testDir, files[2]), []byte("inner file content"), 0664))
		f.Require().NoError(os.Link(path.Join(testDir, files[0]), path.Join(testDir, files[3])))
		f.Require().NoError(os.Symlink(path.Join(testDir, files[0]), path.Join(testDir, files[4])))
		extractDir := path.Join(testDir, "extracted")
		f.Require().NoError(os.Mkdir(extractDir, os.ModePerm))
		f.Require().NoError(exec.Command("tar", append([]string{"--remove-files", "-C", testDir, "-cf", path.Join(testDir, "test.tar")}, files...)...).Run())
		err := f.Extract(path.Join(testDir, "test.tar"), extractDir)

		f.NoError(err)
		f1Buf, err := os.ReadFile(path.Join(extractDir, files[0]))
		f.NoError(err)
		f.Equal(f1Buf, []byte("file content"))
		f.True(f.DoesExist(path.Join(extractDir, files[1])))
		f2Buf, err := os.ReadFile(path.Join(extractDir, files[2]))
		f.NoError(err)
		f.Equal(f2Buf, []byte("inner file content"))
		fileInfo, err := os.Lstat(path.Join(extractDir, files[0]))
		f.NoError(err)
		hardlinkInfo, err := os.Lstat(path.Join(extractDir, files[3]))
		f.NoError(err)
		f.True(os.SameFile(fileInfo, hardlinkInfo))
		symlinkInfo, err := os.Lstat(path.Join(extractDir, files[4]))
		f.NoError(err)
		f.False(os.SameFile(fileInfo, symlinkInfo))
		symlinkDest, err := os.Readlink(path.Join(extractDir, files[4]))
		f.NoError(err)
		f.Equal(path.Join(extractDir, files[0]), symlinkDest)
	})
}
