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

package global

import (
	"sync/atomic"
)

// EventNotifier allows producer that generates many events to notify consumer that event is pending.
// On the other side - consumer always gets the latest event and all previous ones are ignored.
// An event can be of any type.
type EventNotifier[T any] struct {
	ch  chan struct{}
	val atomic.Pointer[T]
}

// NewEventNotifier returns EventNotifier that is ready to be used. If it's not needed anymore it
// must be stopped with Stop() method.
func NewEventNotifier[T any]() *EventNotifier[T] {
	return &EventNotifier[T]{
		ch: make(chan struct{}, 1),
	}
}

// Stop closes notify channel and makes EventNotifier unusable. It should be used by producer.
func (e *EventNotifier[_]) Stop() {
	close(e.ch)
	<-e.ch
}

// GetNotifyChannel returns channels on which consumer gets notifications about new events.
func (e *EventNotifier[_]) GetNotifyChannel() <-chan struct{} {
	return e.ch
}

// GetValue returns latest event that was registered. Consumer should use it after getting
// notification from notify channel.
func (e *EventNotifier[T]) GetValue() *T {
	return e.val.Swap(nil)
}

// Notify should be used by producer to inform consumer about new event.
func (e *EventNotifier[T]) Notify(val T) {
	e.val.Store(&val)

	select {
	case e.ch <- struct{}{}:
	default:
		return
	}
}
