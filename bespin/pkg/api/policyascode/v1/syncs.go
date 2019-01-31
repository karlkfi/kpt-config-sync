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
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Syncs contains all the Sync resources required for Nomos to sync
// Bespin resources to K8S.
var Syncs = []*v1alpha1.Sync{
	{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nomos.dev/v1alpha1",
			Kind:       "Sync",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "bespin",
			Finalizers: []string{v1alpha1.SyncFinalizer},
		},
		Spec: v1alpha1.SyncSpec{
			Groups: []v1alpha1.SyncGroup{
				{
					Group: "bespin.dev",
					Kinds: []v1alpha1.SyncKind{
						{
							Kind: "Folder",
							Versions: []v1alpha1.SyncVersion{
								{
									Version: "v1",
								},
							},
						},
						{
							Kind: "Organization",
							Versions: []v1alpha1.SyncVersion{
								{
									Version: "v1",
								},
							},
						},
						{
							Kind: "Project",
							Versions: []v1alpha1.SyncVersion{
								{
									Version: "v1",
								},
							},
						},
						{
							Kind: "IAMPolicy",
							Versions: []v1alpha1.SyncVersion{
								{
									Version: "v1",
								},
							},
						},
						{
							Kind: "ClusterIAMPolicy",
							Versions: []v1alpha1.SyncVersion{
								{
									Version: "v1",
								},
							},
						},
						{
							Kind: "OrganizationPolicy",
							Versions: []v1alpha1.SyncVersion{
								{
									Version: "v1",
								},
							},
						},
						{
							Kind: "ClusterOrganizationPolicy",
							Versions: []v1alpha1.SyncVersion{
								{
									Version: "v1",
								},
							},
						},
					},
				},
			},
		},
	},
}
