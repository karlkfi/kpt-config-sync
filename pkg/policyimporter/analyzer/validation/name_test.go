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
)

var nameTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.CheckingVisitor {
		return NewNameValidator()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:       "empty",
			Input:      vt.Helper.EmptyRoot(),
			ExpectNoop: true,
		},
		{
			Name:       "acme",
			Input:      vt.Helper.AcmeRoot(),
			ExpectNoop: true,
		},
		{
			Name: "duplicate namespace resource",
			Input: &ast.Root{
				Tree: &ast.TreeNode{
					Objects: vt.ObjectSets(
						vt.Helper.AdminRoleBinding(),
						vt.Helper.AdminRoleBinding(),
					),
				},
			},
			ExpectErr: true,
		},
		{
			Name: "duplicate cluster resource",
			Input: &ast.Root{
				Cluster: &ast.Cluster{
					Objects: vt.ClusterObjectSets(
						vt.Helper.NomosAdminClusterRole(),
						vt.Helper.NomosAdminClusterRole(),
					),
				},
			},
			ExpectErr: true,
		},
	},
}

func TestName(t *testing.T) {
	t.Run("name", nameTestcases.Run)
}
