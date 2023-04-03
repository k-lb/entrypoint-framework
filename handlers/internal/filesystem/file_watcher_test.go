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

	"github.com/fsnotify/fsnotify"
)

func (f *filesystemTestSuite) TestFileWatcher() {
	f.Run("when a directory to a test file does not exist", func() {
		fileWatcher, err := f.NewFileWatcher("not/existing/dir/to/file", fsnotify.Write)
		f.Nil(fileWatcher)
		f.Error(err)
	})

	f.RunWithTestDir("channel capacity and closing it", func(testDir string) {
		testFile := path.Join(testDir, "file.test")
		w, err := f.NewFileWatcher(testFile, fsnotify.Create)
		fw := w.(*FileWatcher)
		f.Require().NoError(err)
		f.Require().NotNil(fw)

		notifier := fw.GetNotificationChannel()
		f.NotNil(notifier)
		f.NotNil(fw.fsnotifyWatcher.Events)
		f.NotNil(fw.fsnotifyWatcher.Errors)
		f.Equal(1, cap(notifier))
		fw.Stop()
		_, open := <-notifier
		f.False(open, "should close an empty notifier channel")
		_, open = <-fw.fsnotifyWatcher.Events
		f.False(open, "should close an empty fsnotify events channel")
		_, open = <-fw.fsnotifyWatcher.Errors
		f.False(open, "should close an empty fsnotify errors channel")
	})

	testCases := [...]struct {
		name             string
		numOfFileChanges int
	}{
		{name: "the file didn't change", numOfFileChanges: 0},
		{name: "the file has changed once", numOfFileChanges: 1},
		{name: "the file has changed multiple times", numOfFileChanges: 100},
	}
	for _, test := range testCases {
		f.RunWithTestDir("when a directory to a test file exists and "+test.name, func(testDir string) {
			done := make(chan bool)
			testFile := path.Join(testDir, "file.test")
			fw, err := f.NewFileWatcher(testFile, fsnotify.Write|fsnotify.Chmod)
			f.Require().NoError(err)
			f.Require().NotNil(fw)
			notifier := fw.GetNotificationChannel()

			for i := 0; i < test.numOfFileChanges; i++ {
				f.writeToFile(testFile)
				_, open := <-notifier
				f.True(open)
				f.Equal(&WatcherEvent{Operation: fsnotify.Write}, fw.GetEvent(), "an event should be returned after file operation")
			}
			f.Nil(fw.GetEvent()) // no new events, previous call should've invalidated value.
			close(done)
		})
	}
}

// writeToFile can not be replaced with os.WriteFile as os.O_TRUNC flag will make extra write events
func (f *filesystemTestSuite) writeToFile(filePath string) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0664)
	f.Require().NoError(err)
	_, err = file.WriteString("content")
	f.Require().NoError(err)
	f.Require().NoError(file.Close())
}
