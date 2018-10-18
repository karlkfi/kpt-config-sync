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

	"k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// rbac allows all this for some reason.
func isChr(c byte) bool {
	return 0 <= c && c != '/' && c != '%' && c <= 127
}

func genRbacName(len int) string {
	b := make([]byte, len)
	a := 0
	var c byte
	for a < len {
		for ; !isChr(c); c++ {
		}
		b[a] = c
		a++
	}
	return string(b)
}

func genName(len int) string {
	name := []string{}
	for i := 0; i < len; i++ {
		name = append(name, "a")
	}
	return strings.Join(name, "")
}

type fakeAPIObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func newFakeAPIObject(m metav1.ObjectMeta) *fakeAPIObject {
	return &fakeAPIObject{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FakeAPIObject",
			APIVersion: "nomos.dev/v1",
		},
		ObjectMeta: m,
	}
}

var metaValidationTestCases = []struct {
	name      string
	obj       metav1.Object
	namespace string
	wantErr   bool
}{
	{
		name:    "nil values in labels / annotations",
		obj:     newFakeAPIObject(metav1.ObjectMeta{}),
		wantErr: false,
	},
	{
		name: "empty metadata",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		}),
		wantErr: false,
	},
	{
		name: "valid label",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Labels:      map[string]string{"foo-corp.com/prod": "true"},
			Annotations: map[string]string{},
		}),
		wantErr: false,
	},
	{
		name: "valid annotation",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{"foo-corp.com/omit-quarks": "up,charm,top"},
		}),
		wantErr: false,
	},
	{
		name: "valid nomos.dev annotation",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{"nomos.dev/namespace-selector": "sre-supported"},
		}),
		wantErr: false,
	},
	{
		name: "invalid label",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Labels:      map[string]string{"nomos.dev/property-xyz": "true"},
			Annotations: map[string]string{},
		}),
		wantErr: true,
	},
	{
		name: "invalid annotation",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Labels:      map[string]string{},
			Annotations: map[string]string{"nomos.dev/property-xyz": "true"},
		}),
		wantErr: true,
	},
	{
		name: "valid name",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Name:        "object-name",
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		}),
		wantErr: false,
	},
	{
		name: "invalid name",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Name:        "A name with invalid chars",
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		}),
		wantErr: true,
	},
	{
		name: "max name len (==253)",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Name:        genName(253),
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		}),
		wantErr: false,
	},
	{
		name: "invalid name len (>253)",
		obj: newFakeAPIObject(metav1.ObjectMeta{
			Name:        genName(254),
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		}),
		wantErr: true,
	},
	{
		name: "rbac Role",
		obj: &v1.Role{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        genRbacName(2048),
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
		namespace: "foo-corp",
		wantErr:   false,
	},
	{
		name: "rbac RoleBinding",
		obj: &v1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        genRbacName(2048),
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
		namespace: "foo-corp",
		wantErr:   false,
	},
	{
		name: "rbac ClusterRole",
		obj: &v1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        genRbacName(2048),
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
		wantErr: false,
	},
	{
		name: "rbac ClusterRoleBinding",
		obj: &v1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRoleBinding",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        genRbacName(2048),
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
		},
		wantErr: false,
	},
}

func init() {
	for i := 0; i < len(metaValidationTestCases); i++ {
		tt := metaValidationTestCases[i]
		if tt.namespace == "" {
			metaValidationTestCases[i].namespace = "test-namespace"
		}
		if tt.obj.GetName() == "" {
			metaValidationTestCases[i].obj.SetName("test-object")
		}
	}
}

func TestValidateObject(t *testing.T) {
	for _, tt := range metaValidationTestCases {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator()

			err := validator.ValidateObject(tt.obj)
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
			obj, ok := tt.obj.(*fakeAPIObject)
			if !ok {
				return
			}

			objValueList := []fakeAPIObject{*obj}
			err := validator.Validate(objValueList)
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
			err = validator.Validate(objPtrList)
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

			err := v.Validate(fakeAPIObjects)
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
		err := v.Validate(nilResList)
		if err != nil {
			t.Errorf("Expected nil, got error: %s", err)
		}
	})
}
