/*
Copyright 2017 The Kubernetes Authors.
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

package namespaceutil

import (
	"testing"

	"github.com/golang/glog"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type isReservedTestcase struct {
	Name   string
	Result bool
}

func TestIsReserved(t *testing.T) {
	for _, testcase := range []isReservedTestcase{
		{"default", true},
		{"kube-public", true},
		{"kube-system", true},
		{"other", false},
		{"name", false},
	} {
		if IsReserved(core_v1.Namespace{ObjectMeta: meta_v1.ObjectMeta{Name: testcase.Name}}) != testcase.Result {
			t.Errorf("Expected %s to have reserved=%t", testcase.Name, testcase.Result)
		}
	}
}

type sanitizeNamespaceTestcase struct {
	Name   string
	Result string
	Panic  bool
}

func panicWrapper(f func() string) (functionPanic bool, result string) {
	defer func() {
		err := recover()
		if err != nil {
			glog.Info("Recovering from err %s", err)
			functionPanic = true
		}
	}()
	result = f()
	return false, result
}

func TestSanitizeNamespace(t *testing.T) {
	for _, testcase := range []sanitizeNamespaceTestcase{
		{"Foo-Bar", "", true},
		{"Foo_Bar", "", true},
		{"-Foo_Bar", "", true},
		{"Foo_Bar-", "", true},
		{"ALL-CAPS", "", true},
		{"foo-bar", "foo-bar", false},
		{"-foo-bar", "", true},
		{"foo-bar-", "", true},
	} {
		panics, sanitized := panicWrapper(func() string { return SanitizeNamespace(testcase.Name) })
		if panics != testcase.Panic {
			t.Errorf("Testcase %#v had unexpected panic=%t", testcase, panics)
		}
		if sanitized != testcase.Result {
			t.Errorf("Testcase %#v had unexpected result %s", testcase, sanitized)
		}
	}
}
