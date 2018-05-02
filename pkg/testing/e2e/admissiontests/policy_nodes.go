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

package admissiontests

import (
	"time"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/e2e/testcontext"
	"github.com/google/nomos/pkg/testing/e2e/testregistry"
	"github.com/pkg/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	testregistry.Register(
		setupFunc,
		nil, // These tests don't require any cleaning up.
		testUnpermittedPolicyNodeDeletions,
		testUnpermittedPolicyNodeUpdates,
		testUnpermittedPolicyNodeCreations,
	)
}

func getPolicyNode(t *testcontext.TestContext, policyNode string) func() error {
	return func() error {
		_, err := t.PolicyHierarchy().NomosV1().PolicyNodes().Get(policyNode, meta_v1.GetOptions{})
		return err
	}
}

func policyNode(name, parent string, policyspace bool) *policyhierarchy_v1.PolicyNode {
	pnt := policyhierarchy_v1.Namespace
	if policyspace {
		pnt = policyhierarchy_v1.Policyspace
	}

	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: name,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Type:   pnt,
			Parent: parent,
		},
	}
}

func setupFunc(t *testcontext.TestContext) {
	t.KubectlApply("examples/acme/policynodes/acme.yaml")

	fn := func(policyNode string) func() error {
		return getPolicyNode(t, policyNode)
	}
	t.WaitForExists(
		time.Second*10,
		fn("acme"),
		fn("eng"),
		fn("backend"),
		fn("frontend"),
		fn("new-prj"),
		fn("newer-prj"),
		fn("rnd"),
	)
}

func testUnpermittedPolicyNodeDeletions(t *testcontext.TestContext) {
	policyNodes := []string{
		"acme",
		"rnd",
	}
	for _, policyNode := range policyNodes {
		err := t.PolicyHierarchy().NomosV1().PolicyNodes().Delete(policyNode, &meta_v1.DeleteOptions{})
		if err == nil {
			panic(errors.Errorf("Unpermitted delete operation on policy node, %s, went through", policyNode))
		}
	}
}

func testUnpermittedPolicyNodeUpdates(t *testcontext.TestContext) {
	policyNodes := []*policyhierarchy_v1.PolicyNode{
		policyNode("foobar-prj", "foobar", false),
		policyNode("acme", "newer-prj", true),
	}
	for _, policyNode := range policyNodes {
		updatedPolicyNode, err := t.PolicyHierarchy().NomosV1().PolicyNodes().Update(policyNode)
		if err == nil {
			panic(errors.Errorf("Unpermitted update operation for policy node: %#v", updatedPolicyNode))
		}
	}
}

func testUnpermittedPolicyNodeCreations(t *testcontext.TestContext) {
	policyNodes := []*policyhierarchy_v1.PolicyNode{
		policyNode("acme", "", true),
		policyNode("eng", "acme", true),
	}
	for _, policyNode := range policyNodes {
		createdPolicyNode, err := t.PolicyHierarchy().NomosV1().PolicyNodes().Create(policyNode)
		if err == nil {
			panic(errors.Errorf("Unpermitted create operation for policy node: %#v", createdPolicyNode))
		}
	}
}
