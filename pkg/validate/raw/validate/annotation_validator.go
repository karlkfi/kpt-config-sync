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

package validate

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	csmetadata "github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
)

// IsInvalidAnnotation returns true if the annotation cannot be declared by users.
func IsInvalidAnnotation(k string) bool {
	return csmetadata.HasConfigSyncPrefix(k) && !csmetadata.IsSourceAnnotation(k)
}

// Annotations verifies that the given object does not have any invalid
// annotations.
func Annotations(obj ast.FileObject) status.Error {
	var invalid []string
	for k := range obj.GetAnnotations() {
		if IsInvalidAnnotation(k) {
			invalid = append(invalid, k)
		}
	}
	if len(invalid) > 0 {
		return metadata.IllegalAnnotationDefinitionError(&obj, invalid)
	}
	return nil
}
