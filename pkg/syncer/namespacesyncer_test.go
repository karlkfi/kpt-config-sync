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

package syncer

import (
	"testing"
	"time"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/client/meta/fake"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/util/workqueue"
)

const testUID = types.UID("0844e3b3-1059-11e8-9233-42010a800005")

type ComputeNamespaceActionsTestCase struct {
	policyNodeNamespaces  []string // namespaces defined in the poicy node objects
	existingNamespaces    []string // namespaces in the active state
	terminatingNamespaces []string // namespaces in the "terminating" state

	needsDelete []string // namespaces that will be deleted
}

func createNamespace(name string, phase core_v1.NamespacePhase) *core_v1.Namespace {
	return &core_v1.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{Name: name},
		Status:     core_v1.NamespaceStatus{Phase: phase},
	}
}

func NewTestNamespaceSyncer() *NamespaceSyncer {
	fakeClient := fake.NewClient()
	kubernetesInformerFactory := informers.NewSharedInformerFactory(
		fakeClient.Kubernetes(), time.Minute)
	return NewNamespaceSyncer(fakeClient, kubernetesInformerFactory.Core().V1().Namespaces().Lister(),
		workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()))
}

func TestSyncerCreate(t *testing.T) {
	syncer := NewTestNamespaceSyncer()
	syncer.OnCreate(&policyhierarchy_v1.PolicyNode{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "test-ns-create",
			UID:  testUID,
		},
	})
	syncer.queue.ShutDown()

	item, _ := syncer.queue.Get()
	action := item.(action.Interface)
	if action.String() != "v1/Namespace/test-ns-create/upsert" {
		t.Errorf("Got unexpected action %s", action.String())
	}
}

func TestSyncerCreatePolicyspace(t *testing.T) {
	syncer := NewTestNamespaceSyncer()
	syncer.OnCreate(&policyhierarchy_v1.PolicyNode{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "test-no-ns-create",
			UID:  testUID,
		},
		Spec: policyhierarchy_v1.PolicyNodeSpec{
			Policyspace: true,
		},
	})
	syncer.queue.ShutDown()

	item, _ := syncer.queue.Get()
	if item != nil {
		t.Errorf("Got unexpected action %s", item)
	}
}

func TestSyncerUpdate(t *testing.T) {
	syncer := NewTestNamespaceSyncer()
	syncer.OnUpdate(
		&policyhierarchy_v1.PolicyNode{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:            "test-ns-update",
				UID:             testUID,
				ResourceVersion: "107",
			},
		},
		&policyhierarchy_v1.PolicyNode{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:            "test-ns-update",
				UID:             testUID,
				ResourceVersion: "139",
			},
		},
	)
	syncer.queue.ShutDown()

	item, _ := syncer.queue.Get()
	action := item.(action.Interface)
	if action.String() != "v1/Namespace/test-ns-update/upsert" {
		t.Errorf("Got unexpected action %s", action.String())
	}
}
