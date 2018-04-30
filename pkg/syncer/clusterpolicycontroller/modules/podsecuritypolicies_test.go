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
// Reviewed by sunilarora
package modules

import (
	"testing"

	"k8s.io/api/extensions/v1beta1"
)

func TestSecurityPoliciesUnpack(t *testing.T) {
	testCases := UnpackTests{
		module: NewPodSecurityPolicies(nil, nil),
		field:  "PodSecurityPoliciesV1Beta1",
		testcases: []UnpackTest{
			UnpackTest{
				name:      "none",
				resources: []string{},
			},
			UnpackTest{
				name:      "one",
				resources: []string{"foo"},
			},
			UnpackTest{
				name:      "two",
				resources: []string{"foo", "bar"},
			},
			UnpackTest{
				name:      "three",
				resources: []string{"foo", "bar", "baz"},
			},
		},
	}
	testCases.Run(t)
}

func TestSecurityPoliciesEqual(t *testing.T) {
	podSecurityPoliciesModule := NewPodSecurityPolicies(nil, nil)
	testCases := []struct {
		name            string
		securityPolicyL *v1beta1.PodSecurityPolicy
		securityPolicyR *v1beta1.PodSecurityPolicy
		wantEqual       bool
	}{
		{
			name:      "nil podsecuritypolicies",
			wantEqual: true,
		},
		{
			name:            "empty podsecuritypolicies",
			securityPolicyL: &v1beta1.PodSecurityPolicy{},
			securityPolicyR: &v1beta1.PodSecurityPolicy{},
			wantEqual:       true,
		},
		{
			name: "different podsecuritypolicies",
			securityPolicyL: &v1beta1.PodSecurityPolicy{
				Spec: v1beta1.PodSecurityPolicySpec{
					Privileged: true,
					SELinux: v1beta1.SELinuxStrategyOptions{
						Rule: v1beta1.SELinuxStrategyRunAsAny,
					},
					SupplementalGroups: v1beta1.SupplementalGroupsStrategyOptions{
						Rule: v1beta1.SupplementalGroupsStrategyRunAsAny,
					},
					RunAsUser: v1beta1.RunAsUserStrategyOptions{
						Rule: v1beta1.RunAsUserStrategyRunAsAny,
					},
					FSGroup: v1beta1.FSGroupStrategyOptions{
						Rule: v1beta1.FSGroupStrategyRunAsAny,
					},
					Volumes: []v1beta1.FSType{
						v1beta1.All,
					},
				},
			},
			securityPolicyR: &v1beta1.PodSecurityPolicy{
				Spec: v1beta1.PodSecurityPolicySpec{
					Privileged: true,
					SELinux: v1beta1.SELinuxStrategyOptions{
						Rule: v1beta1.SELinuxStrategyRunAsAny,
					},
					SupplementalGroups: v1beta1.SupplementalGroupsStrategyOptions{
						Rule: v1beta1.SupplementalGroupsStrategyRunAsAny,
					},
					RunAsUser: v1beta1.RunAsUserStrategyOptions{
						Rule: v1beta1.RunAsUserStrategyMustRunAsNonRoot,
					},
					FSGroup: v1beta1.FSGroupStrategyOptions{
						Rule: v1beta1.FSGroupStrategyRunAsAny,
					},
					Volumes: []v1beta1.FSType{
						v1beta1.All,
					},
				},
			},
			wantEqual: false,
		},
		{
			name: "same podsecuritypolicies",
			securityPolicyL: &v1beta1.PodSecurityPolicy{
				Spec: v1beta1.PodSecurityPolicySpec{
					Privileged: true,
					SELinux: v1beta1.SELinuxStrategyOptions{
						Rule: v1beta1.SELinuxStrategyRunAsAny,
					},
					SupplementalGroups: v1beta1.SupplementalGroupsStrategyOptions{
						Rule: v1beta1.SupplementalGroupsStrategyRunAsAny,
					},
					RunAsUser: v1beta1.RunAsUserStrategyOptions{
						Rule: v1beta1.RunAsUserStrategyRunAsAny,
					},
					FSGroup: v1beta1.FSGroupStrategyOptions{
						Rule: v1beta1.FSGroupStrategyRunAsAny,
					},
					Volumes: []v1beta1.FSType{
						v1beta1.All,
					},
				},
			},
			securityPolicyR: &v1beta1.PodSecurityPolicy{
				Spec: v1beta1.PodSecurityPolicySpec{
					Privileged: true,
					SELinux: v1beta1.SELinuxStrategyOptions{
						Rule: v1beta1.SELinuxStrategyRunAsAny,
					},
					SupplementalGroups: v1beta1.SupplementalGroupsStrategyOptions{
						Rule: v1beta1.SupplementalGroupsStrategyRunAsAny,
					},
					RunAsUser: v1beta1.RunAsUserStrategyOptions{
						Rule: v1beta1.RunAsUserStrategyRunAsAny,
					},
					FSGroup: v1beta1.FSGroupStrategyOptions{
						Rule: v1beta1.FSGroupStrategyRunAsAny,
					},
					Volumes: []v1beta1.FSType{
						v1beta1.All,
					},
				},
			},
			wantEqual: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := podSecurityPoliciesModule.Equal(tt.securityPolicyL, tt.securityPolicyR)
			if got != tt.wantEqual {
				t.Errorf("expected %v got %v", tt.wantEqual, got)
			}
		})
	}
}
