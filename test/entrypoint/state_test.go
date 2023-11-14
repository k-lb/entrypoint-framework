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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateString(t *testing.T) {
	testCases := [...]struct {
		expected string
		state    State
	}{
		{expected: "| inactive | notReady | dead     |", state: State{}},
		{expected: "| inactive | notReady | dead     |", state: State{inactive, notReady, dead}},
		{expected: "| active   | changed  | changing |", state: State{active, changed, changing}},
		{expected: "| active   | updated  | alive    |", state: State{active, updated, alive}},
		{expected: "| active   | applied  | alive    |", state: State{active, applied, alive}},
	}
	for _, test := range testCases {
		assert.Equal(t, test.expected, test.state.string())
	}
}

func TestIsState(t *testing.T) {
	testCases := [...]struct {
		name     string
		state    State
		acts     []ActivationState
		configs  []ConfigurationState
		procs    []ProcessState
		expected bool
	}{
		{name: "when no checking for any state", expected: true},
		{name: "when state is inactive and checking for inactive", state: State{activation: inactive}, acts: []ActivationState{inactive}, expected: true},
		{name: "when state is inactive and checking for inactive or active", state: State{activation: inactive}, acts: []ActivationState{inactive, active}, expected: true},
		{name: "when state is inactive and checking for active", state: State{activation: inactive}, acts: []ActivationState{active}, expected: false},
		{name: "when state is notReady and checking for notReady", state: State{configuration: notReady}, configs: []ConfigurationState{notReady}, expected: true},
		{name: "when state is notReady and checking for notReady or changed", state: State{configuration: notReady}, configs: []ConfigurationState{notReady, changed}, expected: true},
		{name: "when state is notReady and checking for updated or applied", state: State{configuration: notReady}, configs: []ConfigurationState{updated, applied}, expected: false},
		{name: "when state is dead and checking for dead", state: State{process: dead}, procs: []ProcessState{dead}, expected: true},
		{name: "when state is dead and checking for dead or changing", state: State{process: dead}, procs: []ProcessState{dead, changing}, expected: true},
		{name: "when state is dead and checking for changing or alive", state: State{process: dead}, procs: []ProcessState{changing, alive}, expected: false},
		{name: "when state is inactive, notReady, dead and checking for inactive, notReady, dead", state: State{inactive, notReady, dead}, acts: []ActivationState{inactive}, configs: []ConfigurationState{notReady}, procs: []ProcessState{dead}, expected: true},
		{name: "when state is inactive, notReady, dead and checking for inactive or alive, notReady or changed, dead or changing", state: State{inactive, notReady, dead}, acts: []ActivationState{inactive, active}, configs: []ConfigurationState{notReady, changed}, procs: []ProcessState{dead, changing}, expected: true},
		{name: "when state is inactive, notReady, dead and checking for alive, notReady or changed, dead or changing", state: State{inactive, notReady, dead}, acts: []ActivationState{active}, configs: []ConfigurationState{notReady, changed}, procs: []ProcessState{dead, changing}, expected: false},
		{name: "when state is inactive, notReady, dead and checking for inactive or alive, changed or updated, dead or changing", state: State{inactive, notReady, dead}, acts: []ActivationState{inactive, active}, configs: []ConfigurationState{changed, updated}, procs: []ProcessState{dead, changing}, expected: false},
		{name: "when state is inactive, notReady, dead and checking for inactive or alive, notReady or changed, changing or alive", state: State{inactive, notReady, dead}, acts: []ActivationState{inactive, active}, configs: []ConfigurationState{notReady, changed}, procs: []ProcessState{changing, alive}, expected: false},
	}
	for _, test := range testCases {
		inState := is(test.state)
		if len(test.acts) > 0 {
			inState.act(test.acts...)
		}
		if len(test.configs) > 0 {
			inState.config(test.configs...)
		}
		if len(test.procs) > 0 {
			inState.proc(test.procs...)
		}
		assert.Equal(t, test.expected, inState.value(), test.name)
	}
}
