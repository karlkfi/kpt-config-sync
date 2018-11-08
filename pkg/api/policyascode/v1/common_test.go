/*
Copyright 2018 Google LLC.

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

package v1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// These vars are commonly used in unit tests in xxxx_types_test.go.
var fakeImportDetails = ImportDetails{
	Token: "0123456789012345678901234567890123456789",
	Time:  metav1.Date(1998, time.May, 5, 5, 5, 5, 0, time.UTC),
}

var fakeSyncDetails = SyncDetails{
	Error: "",
	Token: "0123456789012345678901234567890123456789",
	Time:  metav1.Date(1998, time.May, 5, 5, 5, 5, 0, time.UTC),
}

var fakeBindings = []IAMPolicyBinding{
	{
		Members: []string{"user:foo", "serviceAccount:bar"},
		Role:    "roles/foo",
	},
	{
		Members: []string{"group:g", "domain:d"},
		Role:    "roles/bar",
	},
}
