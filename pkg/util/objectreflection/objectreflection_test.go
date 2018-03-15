/*
Copyright 2018 The Stolos Authors.
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

package objectreflection

import (
	"testing"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMeta(t *testing.T) {
	orig := &policyhierarchy_v1.PolicyNode{}
	var obj runtime.Object = orig

	aTypeMeta, aObjectMeta := Meta(obj)
	if aTypeMeta != &orig.TypeMeta {
		t.Errorf("TypeMeta pointer mismatch expected: %p actual %p", &orig.TypeMeta, aTypeMeta)
	}
	if aObjectMeta != &orig.ObjectMeta {
		t.Errorf("ObjectMeta pointer mismatch expected: %p actual %p", &orig.ObjectMeta, aObjectMeta)
	}
}

func TestGetNamespaceAndName(t *testing.T) {
	orig := &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "the-namespace",
			Name:      "the-name",
		},
	}

	namespace, name := GetNamespaceAndName(orig)
	if namespace != orig.Namespace {
		t.Errorf("Namespace mismatch %s != %s", orig.Namespace, namespace)
	}
	if name != orig.Name {
		t.Errorf("Name mismatch %s != %s", orig.Name, name)
	}
}
