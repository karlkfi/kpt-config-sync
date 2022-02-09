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
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
)

func TestValidManagementAnnotation(t *testing.T) {
	testCases := []struct {
		name string
		obj  ast.FileObject
		want status.Error
	}{
		{
			name: "no management annotation",
			obj:  fake.Role(),
		},
		{
			name: "disabled management passes",
			obj:  fake.Role(syncertest.ManagementDisabled),
		},
		{
			name: "enabled management fails",
			obj:  fake.Role(syncertest.ManagementEnabled),
			want: fake.Error(nonhierarchical.IllegalManagementAnnotationErrorCode),
		},
		{
			name: "invalid management fails",
			obj:  fake.Role(core.Annotation(metadata.ResourceManagementKey, "invalid")),
			want: fake.Error(nonhierarchical.IllegalManagementAnnotationErrorCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := nonhierarchical.ValidManagementAnnotation(tc.obj)
			if !errors.Is(err, tc.want) {
				t.Errorf("got ValidateCRDName() error %v, want %v", err, tc.want)
			}
		})
	}
}
