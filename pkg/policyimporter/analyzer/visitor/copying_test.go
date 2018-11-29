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

package visitor_test

import (
	"testing"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/visitor"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

var copyingVisitorTestcases = vt.MutatingVisitorTestcases{
	VisitorCtor: func() ast.Visitor {
		return visitor.NewCopying()
	},
	Testcases: []vt.MutatingVisitorTestcase{
		{
			Name:         "empty",
			Input:        vt.Helper.EmptyRoot(),
			ExpectOutput: vt.Helper.EmptyRoot(),
		},
		{
			Name:         "cluster policies",
			Input:        vt.Helper.ClusterPolicies(),
			ExpectOutput: vt.Helper.ClusterPolicies(),
		},
		{
			Name:         "reserved namespaces",
			Input:        vt.Helper.ReservedNamespaces(),
			ExpectOutput: vt.Helper.ReservedNamespaces(),
		},
		{
			Name:         "acme",
			Input:        vt.Helper.AcmeRoot(),
			ExpectOutput: vt.Helper.AcmeRoot(),
		},
	},
}

func TestCopyingVisitor(t *testing.T) {
	t.Run("copyingvisitor", copyingVisitorTestcases.Run)
}
