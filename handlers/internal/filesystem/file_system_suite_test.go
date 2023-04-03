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
	"testing"

	"github.com/stretchr/testify/suite"
)

type filesystemTestSuite struct {
	Filesystem
	suite.Suite
}

func (e *filesystemTestSuite) RunWithTestDir(name string, test func(string)) {
	testDir, err := os.MkdirTemp("", "test_env_*")
	e.Require().NoError(err, "can't build test environment")
	e.Run(name, func() {
		defer os.RemoveAll(testDir) //defer is to make sure it is called even if panic happens
		e.T().Parallel()
		test(testDir)
	})
}

func TestFilesystemTestSuite(t *testing.T) {
	suite.Run(t, &filesystemTestSuite{Filesystem: New(nil)})
}
