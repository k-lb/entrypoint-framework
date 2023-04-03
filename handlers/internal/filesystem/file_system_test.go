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
