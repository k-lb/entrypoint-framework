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

package main

import (
	"io/fs"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"time"
)

const (
	tmp                      = "/tmp"
	watchedActivationPath    = "/tmp/watched/activation/isactive"
	watchedConfigurationPath = "/tmp/watched/configuration/config.tar"
	tmpConfigurationPath     = "/tmp/test.tar"
	numberOfEvents           = 40
)

var isactiveExists = false

// changeActivation deletes watchedActivationPath file if present or create it if it is absent.
func changeActivation() {
	if isactiveExists {
		if err := os.RemoveAll(watchedActivationPath); err != nil {
			panic(err.Error())
		}
		isactiveExists = false
	} else {
		if err := os.WriteFile(watchedActivationPath, []byte{}, os.ModePerm); err != nil {
			panic(err.Error())
		}
		isactiveExists = true
	}
}

// changeConfiguration creates tarred configuration in tmpConfigurationPath with file names and contents from 'files'
// map and moves it to watchedConfigurationPath.
func changeConfiguration(files map[string]string) {
	args := []string{"--remove-files", "-C", tmp, "-cf", tmpConfigurationPath}
	for file, content := range files {
		if err := os.WriteFile(path.Join(tmp, file), []byte(content), 0664); err != nil {
			panic(err.Error())
		}
		args = append(args, file)
	}
	if err := exec.Command("tar", args...).Run(); err != nil {
		panic(err.Error())
	}
	if err := os.Rename(tmpConfigurationPath, watchedConfigurationPath); err != nil {
		panic(err.Error())
	}
}

func main() {
	for _, dir := range [...]string{path.Dir(watchedActivationPath), path.Dir(watchedConfigurationPath)} {
		if err := os.MkdirAll(dir, fs.ModePerm); err != nil {
			panic(err.Error())
		}
	}
	cmd := exec.Command("entrypoint")
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		panic(err.Error())
	}
	go func() {
		log.Printf("entrypoint finished. error: %v\n", cmd.Wait())
	}()
	log.Println("Starting test")
	configs := [...]map[string]string{{"f0": "c0", "f1": "c1", "f2": "c2"}, {"f1": "c1", "f2": "c2.2", "f3": "c3"}}
	for i := 0; i < numberOfEvents; i++ {
		switch rand.Intn(3) {
		case 0:
			log.Printf("activation changing to %t\n", !isactiveExists)
			changeActivation()
		case 1:
			configNum := rand.Intn(2)
			log.Printf("config %d\n", configNum)
			changeConfiguration(configs[configNum])
		case 2:
			log.Printf("waiting for process to end")
			time.Sleep(3 * time.Second / 2)
		}
	}
	if isactiveExists {
		log.Printf("activation changing to false\n")
		changeActivation()
	}
	time.Sleep(time.Second / 100)
}
