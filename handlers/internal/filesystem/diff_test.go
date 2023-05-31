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
	"io/fs"
	"os"
	"path"
)

func (f *filesystemTestSuite) TestAreFilesDifferent() {
	type data struct {
		content string
		mode    fs.FileMode
	}
	testCases := [...]struct {
		name                            string
		firstFile, secondFile           data
		expectedAreDifferent, expectErr bool
	}{
		{name: "when both files do not exist", expectErr: true},
		{name: "when only the first file exists", firstFile: data{"1", 0664}, expectErr: true},
		{name: "when only the second file exists", secondFile: data{"2", 0664}, expectErr: true},
		{name: "when both files exist with different contents", firstFile: data{"diff 1", 0664}, secondFile: data{"diff 2", 0664}, expectedAreDifferent: true},
		{name: "when both files exist with the same content but different modes", firstFile: data{"same", 0775}, secondFile: data{"same", 0664}, expectedAreDifferent: true},
		{name: "when both files exist with the same content and mode", firstFile: data{"same", 0664}, secondFile: data{"same", 0664}},
	}
	for _, test := range testCases {
		test := test
		f.RunWithTestDir(test.name, func(testDir string) {
			firstFilePath := path.Join(testDir, "file0")
			secondFilePath := path.Join(testDir, "file1")
			if test.firstFile.mode != 0 {
				f.Require().NoError(os.WriteFile(firstFilePath, []byte(test.firstFile.content), test.firstFile.mode))
			}
			if test.secondFile.mode != 0 {
				f.Require().NoError(os.WriteFile(secondFilePath, []byte(test.secondFile.content), test.secondFile.mode))
			}
			areDifferent, err := f.AreFilesDifferent(firstFilePath, secondFilePath)

			if test.expectErr {
				f.Error(err)
			} else {
				f.NoError(err)
				f.Equal(test.expectedAreDifferent, areDifferent)
			}
		})
	}
}
