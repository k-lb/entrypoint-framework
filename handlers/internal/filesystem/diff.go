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
	"bytes"
	"os"
)

// AreFilesDifferent returns:
// true and no error if both files can be read and theirs contents or file modes are different,
// false and no error if both files can be read and theirs contents and file modes are the same and
// false and an error if any of files can not be read or status can not be gotten.
func (real) AreFilesDifferent(firstFilePath, secondFilePath string) (bool, error) {
	content1, err := os.ReadFile(firstFilePath)
	if err != nil {
		return false, err
	}
	content2, err := os.ReadFile(secondFilePath)
	if err != nil {
		return false, err
	}
	areContentsDifferent := !bytes.Equal(content1, content2)
	stat1, err := os.Stat(firstFilePath)
	if err != nil {
		return false, err
	}
	stat2, err := os.Stat(secondFilePath)
	if err != nil {
		return false, err
	}
	areFileModesDifferent := stat1.Mode() != stat2.Mode()
	return areContentsDifferent || areFileModesDifferent, nil
}
