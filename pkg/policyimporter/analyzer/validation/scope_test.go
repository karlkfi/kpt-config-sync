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

package validation

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
	ft "github.com/google/nomos/pkg/policyimporter/filesystem/testing"
	"github.com/google/nomos/pkg/policyimporter/meta"
)

func withAPIInfos(root *ast.Root) *ast.Root {
	apiInfos, err := meta.NewAPIInfo(ft.TestAPIResourceList(ft.TestDynamicResources()))
	if err != nil {
		panic("testdata error")
	}
	meta.AddAPIInfo(root, apiInfos)
	return root
}

var scopeTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {

		return NewScope()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:       "empty",
			Input:      withAPIInfos(vt.Helper.EmptyRoot()),
			ExpectNoop: true,
		},
		{
			Name:       "acme",
			Input:      withAPIInfos(vt.Helper.AcmeRoot()),
			ExpectNoop: true,
		},
		{
			Name: "cluster resource at namespace scope",
			Input: withAPIInfos(&ast.Root{
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(vt.Helper.NomosAdminClusterRole()),
				},
			}),
			ExpectErr: true,
		},
		{
			Name: "namespace resource at cluster scope",
			Input: withAPIInfos(&ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(vt.Helper.AdminRoleBinding()),
				},
			}),
			ExpectErr: true,
		},
		{
			Name: "unknown namespace resource",
			Input: withAPIInfos(&ast.Root{
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(vt.Helper.UnknownResource()),
				},
			}),
			ExpectErr: true,
		},
		{
			Name: "unknown cluster resource",
			Input: withAPIInfos(&ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(vt.Helper.UnknownResource()),
				},
			}),
			ExpectErr: true,
		},
	},
}

func TestScope(t *testing.T) {
	t.Run("scope", scopeTestcases.Run)
}
