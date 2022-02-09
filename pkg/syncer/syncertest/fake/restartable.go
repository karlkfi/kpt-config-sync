// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fake

import (
	"github.com/google/nomos/pkg/util/watch"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RestartableManagerRecorder records whether each instance of Restart was
// forced.
type RestartableManagerRecorder struct {
	Restarts []bool
}

// Restart implements watch.RestartableManager.
func (r *RestartableManagerRecorder) Restart(_ map[schema.GroupVersionKind]bool, force bool) (bool, error) {
	r.Restarts = append(r.Restarts, force)
	return false, nil
}

var _ watch.RestartableManager = &RestartableManagerRecorder{}
