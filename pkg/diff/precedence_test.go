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

package diff

import (
	"testing"

	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/diff/difftest"
	"github.com/google/nomos/pkg/syncer/syncertest"
	"github.com/google/nomos/pkg/testing/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCanManage(t *testing.T) {
	testCases := []struct {
		name       string
		reconciler declared.Scope
		object     client.Object
		want       bool
	}{
		{
			"Root can manage unmanaged object",
			declared.RootReconciler,
			fake.DeploymentObject(),
			true,
		},
		{
			"Root can manage other-managed object",
			declared.RootReconciler,
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedBy("foo")),
			true,
		},
		{
			"Root can manage self-managed object",
			declared.RootReconciler,
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedByRoot),
			true,
		},
		{
			"Non-root can manage unmanaged object",
			"foo",
			fake.DeploymentObject(),
			true,
		},
		{
			"Non-root can manage self-managed object",
			"foo",
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedBy("foo")),
			true,
		},
		{
			"Non-root can manage other-managed object",
			"foo",
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedBy("foo")),
			true,
		},
		{
			"Non-root can NOT manage root-managed object",
			"foo",
			fake.DeploymentObject(syncertest.ManagementEnabled, difftest.ManagedByRoot),
			false,
		},
		{
			"Non-root can manage seemingly root-managed object",
			"foo",
			fake.DeploymentObject(difftest.ManagedByRoot),
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanManage(tc.reconciler, tc.object)
			if got != tc.want {
				t.Errorf("CanManage() = %v; want %v", got, tc.want)
			}
		})
	}
}
