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

package applier

import (
	"fmt"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
)

// ApplierErrorCode is the error code for apply failures.
const ApplierErrorCode = "2009"

var applierErrorBuilder = status.NewErrorBuilder(ApplierErrorCode)

// Error indicates that the applier failed to apply some resources.
func Error(err error) status.Error {
	return applierErrorBuilder.Wrap(err).Build()
}

// ErrorForResource indicates that the applier filed to apply
// the given resource.
func ErrorForResource(err error, id core.ID) status.Error {
	return applierErrorBuilder.Wrap(fmt.Errorf("failed to apply %v: %w", id, err)).Build()
}
