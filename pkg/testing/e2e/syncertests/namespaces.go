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

package syncertests

import (
	"time"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/testing/e2e/testcontext"
	"github.com/google/nomos/pkg/testing/e2e/testregistry"
	"github.com/pkg/errors"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testSyncerNamespaces = "test-syncer-namespaces"
)

var testSyncerNamespacesPolicyNode = policyhierarchy_v1.PolicyNode{
	ObjectMeta: meta_v1.ObjectMeta{
		Name: testSyncerNamespaces,
	},
	Spec: policyhierarchy_v1.PolicyNodeSpec{
		Policyspace: false,
		Parent:      "eng",
	},
}

func init() {
	testregistry.Register(
		setupFunc,
		nil,
		testNamespaceGarbageCollection,
	)
}

func getNamespace(t *testcontext.TestContext, namespace string) func() error {
	return func() error {
		_, err := t.Kubernetes().CoreV1().Namespaces().Get(namespace, meta_v1.GetOptions{})
		return err
	}
}

func setupFunc(t *testcontext.TestContext) {
	t.KubectlApply("examples/acme/policynodes/acme.yaml")

	// TODO: This should probably check syncer status rather than namespace existence.
	fn := func(namespace string) func() error {
		return getNamespace(t, namespace)
	}
	t.WaitForExists(time.Second*10, fn("backend"), fn("frontend"), fn("new-prj"), fn("newer-prj"))
}

// TODO(briantkennedy)
// nolint: deadcode, errcheck, megacheck
func cleanupFunc(t *testcontext.TestContext) {
	t.PolicyHierarchy().NomosV1().PolicyNodes().Delete(testSyncerNamespaces, &meta_v1.DeleteOptions{})
	t.Kubernetes().CoreV1().Namespaces().Delete(testSyncerNamespaces, &meta_v1.DeleteOptions{})
}

func testNamespaceGarbageCollection(t *testcontext.TestContext) {
	_, err := t.PolicyHierarchy().NomosV1().PolicyNodes().Create(
		testSyncerNamespacesPolicyNode.DeepCopy())
	if err != nil && !api_errors.IsAlreadyExists(err) {
		panic(errors.Wrapf(err, "Failed to create policy node"))
	}

	t.WaitForExists(time.Second*10, func() error {
		_, err := t.Kubernetes().CoreV1().Namespaces().Get(testSyncerNamespaces, meta_v1.GetOptions{})
		return err
	})

	propagationPolicy := meta_v1.DeletePropagationForeground
	// TODO(briantkennedy): Handle errors?
	// nolint: errcheck
	t.PolicyHierarchy().NomosV1().PolicyNodes().Delete(testSyncerNamespaces, &meta_v1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})

	t.WaitForNotFound(time.Second*10, func() error {
		_, err := t.PolicyHierarchy().NomosV1().PolicyNodes().Get(
			testSyncerNamespaces, meta_v1.GetOptions{})
		return err
	})
}
