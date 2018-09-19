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

package action

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObjectMetaTestCase struct {
	Name         string
	Left         metav1.ObjectMeta
	Right        metav1.ObjectMeta
	ExpectReturn bool
}

func newObjectMeta(labels map[string]string, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Labels:      labels,
		Annotations: annotations,
	}
}

var objectMetaSubsetTestcases = []ObjectMetaTestCase{
	ObjectMetaTestCase{
		Name: "labels and annotations are both subsets",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: true,
	},
	ObjectMetaTestCase{
		Name: "labels not subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	ObjectMetaTestCase{
		Name: "annotations not subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k5": "v5"},
		),
		ExpectReturn: false,
	},
	ObjectMetaTestCase{
		Name: "neither are subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k6": "v6"},
		),
		ExpectReturn: false,
	},
}

func TestObjectMetaSubset(t *testing.T) {
	for _, tc := range objectMetaSubsetTestcases {
		t.Run(tc.Name, func(t *testing.T) {
			set := &rbacv1.Role{ObjectMeta: tc.Left}
			subset := &rbacv1.Role{ObjectMeta: tc.Right}
			if ObjectMetaSubset(set, subset) != tc.ExpectReturn {
				t.Fatalf("Testcase Failure %v", tc)
			}
		})
	}
}

var objectMetaEqualTestcases = []ObjectMetaTestCase{
	ObjectMetaTestCase{
		Name: "labels and annotations are qual",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		ExpectReturn: true,
	},
	ObjectMetaTestCase{
		Name:         "labels and annotations not set",
		Left:         newObjectMeta(nil, nil),
		Right:        newObjectMeta(nil, nil),
		ExpectReturn: true,
	},
	ObjectMetaTestCase{
		Name: "labels and annotations are subsets",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	ObjectMetaTestCase{
		Name: "neither are subset",
		Left: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k6": "v6"},
		),
		ExpectReturn: false,
	},
	ObjectMetaTestCase{
		Name: "left labels and annotations not set",
		Left: newObjectMeta(nil, nil),
		Right: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	ObjectMetaTestCase{
		Name: "right labels and annotations not set",
		Left: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		Right:        newObjectMeta(nil, nil),
		ExpectReturn: false,
	},
}

func TestObjectMetaEqual(t *testing.T) {
	for _, tc := range objectMetaEqualTestcases {
		t.Run(tc.Name, func(t *testing.T) {
			left := &rbacv1.Role{ObjectMeta: tc.Left}
			right := &rbacv1.Role{ObjectMeta: tc.Right}
			if ObjectMetaEqual(left, right) != tc.ExpectReturn {
				t.Fatalf("Testcase Failure %v", tc)
			}
		})
	}
}
