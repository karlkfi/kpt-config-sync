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
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepoSyncObjectV1Alpha1 initializes a RepoSync with version v1alpha1.
func RepoSyncObjectV1Alpha1(ns, name string, opts ...core.MetaMutator) *v1alpha1.RepoSync {
	result := &v1alpha1.RepoSync{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		TypeMeta: ToTypeMeta(kinds.RepoSyncV1Alpha1()),
	}
	mutate(result, opts...)

	return result
}

// RepoSyncObjectV1Beta1 initializes a RepoSync with version v1beta1.
func RepoSyncObjectV1Beta1(ns, name string, opts ...core.MetaMutator) *v1beta1.RepoSync {
	result := &v1beta1.RepoSync{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		TypeMeta: ToTypeMeta(kinds.RepoSyncV1Beta1()),
	}
	mutate(result, opts...)

	return result
}
