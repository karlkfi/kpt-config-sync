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

package meta

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func genName(len int) string {
	name := []string{}
	for i := 0; i < len; i++ {
		name = append(name, "a")
	}
	return strings.Join(name, "")
}

func roleBindingMaxLen(namespace string) int {
	return 253 - len(namespace) - 1
}

var metaValidationTestCases = []struct {
	name      string
	meta      metav1.ObjectMeta
	flattened bool
	namespace string
	wantErr   bool
}{
	{
		name:    "nil values in labels / annotations",
		meta:    metav1.ObjectMeta{},
		wantErr: false,
	},
	{
		name: "empty metadata",
		meta: metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		wantErr: false,
	},
	{
		name: "valid label",
		meta: metav1.ObjectMeta{
			Labels:      map[string]string{"foo-corp.com/prod": "true"},
			Annotations: map[string]string{},
		},
		wantErr: false,
	},
	{
		name: "valid annotation",
		meta: metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{"foo-corp.com/omit-quarks": "up,charm,top"},
		},
		wantErr: false,
	},
	{
		name: "invalid label",
		meta: metav1.ObjectMeta{
			Labels:      map[string]string{"nomos.dev/property-xyz": "true"},
			Annotations: map[string]string{},
		},
		wantErr: true,
	},
	{
		name: "invalid annotation",
		meta: metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{"nomos.dev/property-xyz": "true"},
		},
		wantErr: true,
	},
	{
		name: "valid name",
		meta: metav1.ObjectMeta{
			Name:        "object-name",
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		wantErr: false,
	},
	{
		name: "invalid name",
		meta: metav1.ObjectMeta{
			Name:        "A name with invalid chars",
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		wantErr: true,
	},
	{
		name: "max name len (==253)",
		meta: metav1.ObjectMeta{
			Name:        genName(253),
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		wantErr: false,
	},
	{
		name: "invalid name len (>253)",
		meta: metav1.ObjectMeta{
			Name:        genName(254),
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		wantErr: true,
	},
	{
		name: "valid hierarchical name len (=253)",
		meta: metav1.ObjectMeta{
			Name:        genName(roleBindingMaxLen("foo-corp")),
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		flattened: true,
		namespace: "foo-corp",
		wantErr:   false,
	},
	{
		name: "valid hierarchical name len (>253)",
		meta: metav1.ObjectMeta{
			Name:        genName(roleBindingMaxLen("foo-corp") + 1),
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		flattened: true,
		namespace: "foo-corp",
		wantErr:   true,
	},
}

func init() {
	for i := 0; i < len(metaValidationTestCases); i++ {
		tt := metaValidationTestCases[i]
		if tt.namespace == "" {
			metaValidationTestCases[i].namespace = "test-namespace"
		}
		if tt.meta.Name == "" {
			metaValidationTestCases[i].meta.Name = "test-object"
		}
	}
}

type fakeAPIObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func TestValidateObject(t *testing.T) {
	for _, tt := range metaValidationTestCases {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator()
			validator.Flattened = tt.flattened
			obj := &fakeAPIObject{
				TypeMeta: metav1.TypeMeta{
					Kind:       "FakeAPIObject",
					APIVersion: "nomos.dev/v1",
				},
				ObjectMeta: tt.meta,
			}

			err := validator.ValidateObject(tt.namespace, obj)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected nil, got error: %s", err)
				}
			}
		})
	}
}

// Run TestValidateObject testcases through Validate function.
func TestValidateMetaChecks(t *testing.T) {
	for _, tt := range metaValidationTestCases {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator()
			validator.Flattened = tt.flattened
			obj := &fakeAPIObject{
				TypeMeta: metav1.TypeMeta{
					Kind:       "FakeAPIObject",
					APIVersion: "nomos.dev/v1",
				},
				ObjectMeta: tt.meta,
			}

			objValueList := []fakeAPIObject{*obj}
			err := validator.Validate(tt.namespace, objValueList)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected nil, got error: %s", err)
				}
			}

			objPtrList := []*fakeAPIObject{obj}
			err = validator.Validate(tt.namespace, objPtrList)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected nil, got error: %s", err)
				}
			}
		})
	}
}

var uniqueNamesTestCases = []struct {
	name          string
	resourceNames []string
	wantErr       bool
}{
	{
		name:          "no values",
		resourceNames: []string{},
		wantErr:       false,
	},
	{
		name:          "one value",
		resourceNames: []string{"foo"},
		wantErr:       false,
	},
	{
		name:          "two values",
		resourceNames: []string{"foo", "test"},
		wantErr:       false,
	},
	{
		name:          "three values",
		resourceNames: []string{"foo", "xyz", "test"},
		wantErr:       false,
	},
	{
		name:          "two with duplicate",
		resourceNames: []string{"foo", "foo"},
		wantErr:       true,
	},
	{
		name:          "three with duplicate",
		resourceNames: []string{"foo", "test", "foo"},
		wantErr:       true,
	},
	{
		name:          "four with duplicate",
		resourceNames: []string{"foo", "xyz", "test", "foo"},
		wantErr:       true,
	},
}

func TestValidateUniqueNames(t *testing.T) {
	for _, tt := range uniqueNamesTestCases {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()

			fakeAPIObjects := []*fakeAPIObject{}
			for _, name := range tt.resourceNames {
				fakeAPIObjects = append(fakeAPIObjects, &fakeAPIObject{
					TypeMeta: metav1.TypeMeta{
						Kind:       "FakeAPIObject",
						APIVersion: "nomos.dev/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
					},
				})
			}

			err := v.Validate("foo-corp", fakeAPIObjects)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected nil, got error: %s", err)
				}
			}
		})
	}

	t.Run("check nil list", func(t *testing.T) {
		v := NewValidator()
		var nilResList []fakeAPIObject
		err := v.Validate("foo-corp", nilResList)
		if err != nil {
			t.Errorf("Expected nil, got error: %s", err)
		}
	})
}
