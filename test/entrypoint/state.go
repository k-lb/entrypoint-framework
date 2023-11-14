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
	"fmt"
	"slices"
)

// ActivationState represents activation state of the system.
type ActivationState bool

const (
	inactive ActivationState = false
	active   ActivationState = true
)

// ConfigurationState represents configuration state of the system.
type ConfigurationState int

const (
	notReady ConfigurationState = iota
	changed
	updated
	applied
)

// ProcessState represents process state of the system.
type ProcessState int

const (
	dead ProcessState = iota
	changing
	alive
)

// State represents a current state of the system.
type State struct {
	activation    ActivationState
	configuration ConfigurationState
	process       ProcessState
}

// string returns string representation of a State.
func (s State) string() string {
	activation := "inactive"
	if s.activation == active {
		activation = "active"
	}
	configuration := "notReady"
	switch s.configuration {
	case changed:
		configuration = "changed"
	case updated:
		configuration = "updated"
	case applied:
		configuration = "applied"
	}
	process := "dead"
	switch s.process {
	case changing:
		process = "changing"
	case alive:
		process = "alive"
	}
	return fmt.Sprintf("| %-8s | %-8s | %-8s |", activation, configuration, process)
}

// InState is a helper struct for checking a State. It should be used like
// is(state).act(a, ...).config(c, ...).proc(p, ...).value() which is equivalent to
// (state.activation == a || ...) && (state.configuration == p || ...) && (state.process == p || ...)
type InState struct {
	State
	isState bool
}

// is creates a new InState pointer with isState true.
func is(s State) *InState {
	return &InState{s, true}
}

// setFalseWhenMissing sets isState to false if val is not in list and returns InState pointer. If i is nil returns nil.
func setFalseWhenMissing[T comparable](i *InState, val T, list ...T) *InState {
	if i == nil {
		return i
	}
	if !slices.Contains(list, val) {
		i.isState = false
	}
	return i
}

// act set to false isState if State.activation is missing in activations.
func (i *InState) act(activations ...ActivationState) *InState {
	return setFalseWhenMissing(i, i.State.activation, activations...)
}

// config set to false isState if State.configuration is missing in configurations.
func (i *InState) config(configurations ...ConfigurationState) *InState {
	return setFalseWhenMissing(i, i.State.configuration, configurations...)
}

// proc set to false isState if State.process is missing in processes.
func (i *InState) proc(processes ...ProcessState) *InState {
	return setFalseWhenMissing(i, i.State.process, processes...)
}

// value returns isState bool value.
func (i *InState) value() bool {
	return i.isState
}
