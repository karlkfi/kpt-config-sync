/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policyhierarchycontroller

import (
	"testing"
	"time"
)

type warningFilterEvent struct {
	name       string
	event      string
	delta      time.Duration
	expectWarn bool
}

type warningFilterTestcase struct {
	name   string
	count  int64         // default to 3
	delta  time.Duration // default to 60 seconds
	events []warningFilterEvent

	fakeNow time.Time
}

func (tt *warningFilterTestcase) fakeTimeNow() time.Time {
	return tt.fakeNow
}

func (tt *warningFilterTestcase) Run(t *testing.T) {
	if tt.count == 0 {
		tt.count = 3
	}
	if tt.delta == 0 {
		tt.delta = time.Minute
	}
	tt.fakeNow = time.Now()

	wf := NewWarningFilter(tt.count, tt.delta)
	wf.timeNow = tt.fakeTimeNow
	for idx, event := range tt.events {
		tt.fakeNow = tt.fakeNow.Add(event.delta)

		var warning string
		switch event.event {
		case "", "warning":
			warning = wf.Warning(event.name)
		case "clear":
			wf.Clear(event.name)
		}

		if event.expectWarn {
			if warning == "" {
				t.Errorf("expected warning on event index %d but got none", idx)
			}
		} else {
			if warning != "" {
				t.Errorf("expected no warning on event index %d but got %v", idx, warning)
			}
		}
	}
}

var warningFilterTestcases = []warningFilterTestcase{
	{
		name: "Exceed count but not time",
		events: []warningFilterEvent{
			{name: "foo"},
			{name: "foo", delta: time.Second},
			{name: "foo", delta: time.Second},
		},
	},
	{
		name: "Exceed time but not count",
		events: []warningFilterEvent{
			{name: "foo"},
			{name: "foo", delta: time.Second * 120},
		},
	},
	{
		name: "Exceed time and count",
		events: []warningFilterEvent{
			{name: "foo"},
			{name: "foo", delta: time.Second * 30},
			{name: "foo", delta: time.Second * 31, expectWarn: true},
		},
	},
	{
		name: "Clear then warning",
		events: []warningFilterEvent{
			{name: "foo"},
			{name: "foo", delta: time.Second * 30},
			{name: "foo", event: "clear"},
			{name: "foo", delta: time.Second * 31},
		},
	},
}

func TestWarningFilter(t *testing.T) {
	for _, tt := range warningFilterTestcases {
		t.Run(tt.name, tt.Run)
	}
}
