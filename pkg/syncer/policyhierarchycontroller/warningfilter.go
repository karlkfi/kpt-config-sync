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
	"fmt"
	"sync"
	"time"
)

type resource struct {
	enterTime time.Time
	count     int64
}

// WarningFilter will suppress a warning until a both time AND count threshold have been exceeded.
type WarningFilter struct {
	countThreshold int64
	timeThreshold  time.Duration
	resources      map[string]*resource
	timeNow        func() time.Time
	mutex          sync.Mutex
}

// NewWarningFilter creates a new WarningFilter with the specified count and time thresholds.
func NewWarningFilter(countThreshold int64, timeThreshold time.Duration) *WarningFilter {
	return &WarningFilter{
		countThreshold: countThreshold,
		timeThreshold:  timeThreshold,
		resources:      make(map[string]*resource),
		timeNow:        time.Now,
	}
}

// Warning registers a warning and returns a non-emtpy string if the warning should be emitted.
func (d *WarningFilter) Warning(name string) string {
	timeNow := d.timeNow()
	d.mutex.Lock()
	r, found := d.resources[name]
	if !found {
		r = &resource{
			enterTime: timeNow,
		}
		d.resources[name] = r
	}
	r.count++
	curCount := r.count
	elapsed := timeNow.Sub(r.enterTime)
	d.mutex.Unlock()

	if curCount < d.countThreshold {
		return ""
	}
	if elapsed < d.timeThreshold {
		return ""
	}

	return fmt.Sprintf(
		"resource %q has been in error state for more than %v tries and %v",
		name,
		r.count,
		elapsed,
	)
}

// Clear removes the item from state tracking
func (d *WarningFilter) Clear(name string) {
	delete(d.resources, name)
}
