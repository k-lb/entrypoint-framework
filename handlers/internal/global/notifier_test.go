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

func (s *eventNotifierTestSuite) TestGetNotifyChannel() {
	ch := s.en.GetNotifyChannel()
	s.Assert().Equal(1, cap(ch))
	s.Assert().Empty(ch)
}

func (s *eventNotifierTestSuite) TestGetValue() {
	foo := "foo"
	s.Assert().Zero(s.en.GetValue())
	s.en.val.Store(&foo)
	s.Assert().Equal(s.en.val.Load(), s.en.GetValue())
	s.Nil(s.en.GetValue()) // previous call to GetValue shoud've set invalidate value
}

func (s *eventNotifierTestSuite) TestNotify() {
	tests := [...]struct {
		name string
		vals []string
		idx  int
	}{
		{name: "empty", vals: []string{}, idx: -1},
		{name: "one event", vals: []string{"foo"}, idx: -1},
		{name: "few events", vals: []string{"foo", "bar", "baz", "qux", "quux", "pizza"}, idx: 2},
	}
	for _, test := range tests {
		test := test
		s.Run(test.name, func() {
			for i, v := range test.vals {
				s.en.Notify(v)
				if i == test.idx {
					val := s.en.GetValue()
					s.NotNil(val)
					s.Equal(v, *val)
				}
			}
			l := len(test.vals)
			if l == 0 {
				s.Nil(s.en.GetValue())
			} else {
				s.Equal(test.vals[l-1], *s.en.GetValue())
			}
		})
	}
}
