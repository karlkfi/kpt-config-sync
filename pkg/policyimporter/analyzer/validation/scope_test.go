/*
Copyright 2017 The Nomos Authors.
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

package validation_test

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/transform/tree/treetesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	ft "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	"github.com/google/nomos/pkg/util/discovery"
	"k8s.io/apimachinery/pkg/runtime"
)

func withPath(o runtime.Object, path string) ast.FileObject {
	return ast.FileObject{
		Object:   o,
		Relative: nomospath.NewRelative(path),
	}
}

func withScope(r *ast.Root) *ast.Root {
	apiInfos, err := discovery.NewAPIInfo(ft.TestAPIResourceList(ft.TestDynamicResources()))
	if err != nil {
		panic("testdata error")
	}
	discovery.AddAPIInfo(r, apiInfos)
	return r
}

func TestScope(t *testing.T) {
	var scopeTestcases = vt.MutatingVisitorTestcases{
		VisitorCtor: func() ast.Visitor {
			return validation.NewScope()
		},
		Testcases: []vt.MutatingVisitorTestcase{
			{
				Name:       "empty",
				Input:      withScope(vt.Helper.EmptyRoot()),
				ExpectNoop: true,
			},
			{
				Name:       "acme",
				Input:      withScope(vt.Helper.AcmeRoot()),
				ExpectNoop: true,
			},
			{
				Name:      "cluster resource at namespace scope",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.NomosAdminClusterRole(), "namespaces/cr.yaml"))),
				ExpectErr: true,
			},
			{
				Name:       "cluster resource at cluster scope",
				Input:      withScope(treetesting.BuildTree(t, withPath(vt.Helper.NomosAdminClusterRole(), "cluster/cr.yaml"))),
				ExpectNoop: true,
			},
			{
				Name:      "namespace resource at cluster scope",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.AdminRoleBinding(), "cluster/cr.yaml"))),
				ExpectErr: true,
			},
			{
				Name:       "namespace resource at namespace scope",
				Input:      withScope(treetesting.BuildTree(t, withPath(vt.Helper.AdminRoleBinding(), "namespaces/cr.yaml"))),
				ExpectNoop: true,
			},
			{
				Name:      "unknown namespace resource",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.UnknownResource(), "namespaces/cr.yaml"))),
				ExpectErr: true,
			},
			{
				Name:      "unknown cluster resource",
				Input:     withScope(treetesting.BuildTree(t, withPath(vt.Helper.UnknownResource(), "cluster/cr.yaml"))),
				ExpectErr: true,
			},
		},
	}
	t.Run("scope", scopeTestcases.Run)
}
