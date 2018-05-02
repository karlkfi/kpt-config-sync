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

	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ObjectMetaSubsetTestcase struct {
	Name         string
	Set          meta_v1.ObjectMeta
	Subset       meta_v1.ObjectMeta
	ExpectReturn bool
}

func newObjectMeta(labels map[string]string, annotations map[string]string) meta_v1.ObjectMeta {
	return meta_v1.ObjectMeta{
		Labels:      labels,
		Annotations: annotations,
	}
}

func (tc *ObjectMetaSubsetTestcase) Run(t *testing.T) {
	var Set, Subset runtime.Object
	Set = &rbac_v1.Role{ObjectMeta: tc.Set}
	Subset = &rbac_v1.Role{ObjectMeta: tc.Subset}
	if ObjectMetaSubset(Set, Subset) != tc.ExpectReturn {
		t.Errorf("Testcase Failure %v", tc)
	}
}

var objectMetaTestcases = []ObjectMetaSubsetTestcase{
	ObjectMetaSubsetTestcase{
		Name: "labels and annotations are both subsets",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: true,
	},
	ObjectMetaSubsetTestcase{
		Name: "labels not subset",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k3": "v3"},
		),
		ExpectReturn: false,
	},
	ObjectMetaSubsetTestcase{
		Name: "annotations not subset",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k1": "v1"},
			map[string]string{"k5": "v5"},
		),
		ExpectReturn: false,
	},
	ObjectMetaSubsetTestcase{
		Name: "neither are subset",
		Set: newObjectMeta(
			map[string]string{"k1": "v1", "k2": "v2"},
			map[string]string{"k3": "v3", "k4": "v4"},
		),
		Subset: newObjectMeta(
			map[string]string{"k5": "v5"},
			map[string]string{"k6": "v6"},
		),
		ExpectReturn: false,
	},
}

func TestObjectMetaSubset(t *testing.T) {
	for _, testcase := range objectMetaTestcases {
		t.Run(testcase.Name, testcase.Run)
	}
}

type IsSubsetTestcase struct {
	Name         string
	Set          map[string]string
	Subset       map[string]string
	ExpectReturn bool
}

func (tc *IsSubsetTestcase) Run(t *testing.T) {
	if isSubset(tc.Set, tc.Subset) != tc.ExpectReturn {
		t.Errorf("Testcase Failure %v", tc)
	}
}

var isSubsetTestcases = []IsSubsetTestcase{
	IsSubsetTestcase{
		Name:         "both nil/empty",
		Set:          nil,
		Subset:       nil,
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "subset nil/empty",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       nil,
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "set nil/empty",
		Set:          nil,
		Subset:       map[string]string{"k1": "v1", "k2": "v2"},
		ExpectReturn: false,
	},
	IsSubsetTestcase{
		Name:         "subset is subset",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1"},
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "both equivalent",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1", "k2": "v2"},
		ExpectReturn: true,
	},
	IsSubsetTestcase{
		Name:         "subset has extra elements equivalent",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
		ExpectReturn: false,
	},
	IsSubsetTestcase{
		Name:         "subset is not subset",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k3": "v3"},
		ExpectReturn: false,
	},
	IsSubsetTestcase{
		Name:         "subset has different value",
		Set:          map[string]string{"k1": "v1", "k2": "v2"},
		Subset:       map[string]string{"k1": "v1", "k2": "mismatch"},
		ExpectReturn: false,
	},
}

func TestIsSubset(t *testing.T) {
	for _, testcase := range isSubsetTestcases {
		t.Run(testcase.Name, testcase.Run)
	}
}
