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
package policynodewatcher

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/golang/glog"

	policyhierarchy_v1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy/fake"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	api_testing "k8s.io/client-go/testing"
)

// testEvent is an event received during the testcase
type testEvent struct {
	eventType  EventType
	policyNode *policyhierarchy_v1.PolicyNode
}

type PolicyNodeWatcherTestHelper struct {
	FakeClient      *fake.Clientset
	Watcher         *PolicyNodeWatcher
	resourceVersion int

	// Records for callbacks
	// TestEvents is a list of events received from OnEvent
	TestEvents []*testEvent
	// ErrValue is the error recieved from OnError
	ErrValue error
	// CbCount is the total number of OnEvent/OnError calls
	CbCount int

	// The list of all operatiosn that will be performed in the testcase.  A nil event represents
	// closing the event channel.
	testOperations []*watch.Event
	// The operation queue, this will be mutated as events are generated.
	testOperationQueue []*watch.Event

	// The StopCheck function, this callback will be invoked in each OnEvent/OnError call so that
	// the testcase can control when to stop watching.
	StopCheck func()
}

var _ EventHandler = &PolicyNodeWatcherTestHelper{}

func (h *PolicyNodeWatcherTestHelper) OnEvent(eventType EventType, policyNode *policyhierarchy_v1.PolicyNode) {
	glog.Info("Got policy node callback")
	h.TestEvents = append(h.TestEvents, &testEvent{
		eventType:  eventType,
		policyNode: policyNode,
	})
	h.CbCount++
	if h.StopCheck != nil {
		h.StopCheck()
	}
}

func (h *PolicyNodeWatcherTestHelper) OnError(err error) {
	glog.Infof("Got error callback: %s", err)
	h.ErrValue = err
	h.CbCount++
	if h.StopCheck != nil {
		h.StopCheck()
	}
}

func (h *PolicyNodeWatcherTestHelper) React(action api_testing.Action) (handled bool, ret watch.Interface, err error) {
	if action.GetResource().Group != policyhierarchy_v1.GroupName {
		return false, nil, nil
	}

	glog.Info("In watch reactor for %s", spew.Sdump(action))

	fakeWatcher := watch.NewFake()

	go func() {
		for {
			if len(h.testOperationQueue) == 0 {
				glog.Infof("No more operations, fake watcher stopping")
				fakeWatcher.Stop()
				return
			}

			op := h.testOperationQueue[0]
			h.testOperationQueue = h.testOperationQueue[1:]
			if op == nil {
				glog.Infof("Fake watcher simulating CLOSE")
				fakeWatcher.Stop()
				return
			}

			glog.Infof("Fake watcher simulating %s", op.Type)
			fakeWatcher.Action(op.Type, op.Object)
		}
	}()

	return true, fakeWatcher, nil
}

func (h *PolicyNodeWatcherTestHelper) NextResourceVersion() string {
	// simulate non-sequential resource version
	h.resourceVersion += rand.Int() % 5
	return fmt.Sprintf("%d", h.resourceVersion)
}

func (h *PolicyNodeWatcherTestHelper) ResourceVersion() string {
	return fmt.Sprintf("%d", h.resourceVersion)
}

func (h *PolicyNodeWatcherTestHelper) ActionAdd() {
	h.action(watch.Added)
}

func (h *PolicyNodeWatcherTestHelper) ActionModify() {
	h.action(watch.Modified)
}

func (h *PolicyNodeWatcherTestHelper) ActionDelete() {
	h.action(watch.Deleted)
}

func (h *PolicyNodeWatcherTestHelper) ActionError() {
	h.queueEvent(&watch.Event{Type: watch.Error, Object: nil})
}

func (h *PolicyNodeWatcherTestHelper) ActionClose() {
	h.queueEvent(nil)
}

func (h *PolicyNodeWatcherTestHelper) action(eventType watch.EventType) {
	h.queueEvent(&watch.Event{
		Type: eventType,
		Object: &policyhierarchy_v1.PolicyNode{
			ObjectMeta: meta_v1.ObjectMeta{
				ResourceVersion: h.NextResourceVersion(),
			},
		},
	})
}

func (h *PolicyNodeWatcherTestHelper) queueEvent(event *watch.Event) {
	h.testOperations = append(h.testOperations, event)
	h.testOperationQueue = h.testOperations
}

func (h *PolicyNodeWatcherTestHelper) DefaultStopper() func() {
	ops := 0
	hasError := false
	for _, val := range h.testOperations {
		if val != nil {
			ops++
			if val.Type == watch.Error {
				hasError = true
			}
		}
	}
	glog.Infof("Creating default stopper for %d actions, error %t", ops, hasError)
	return func() {
		if h.CbCount >= ops && !hasError {
			h.Watcher.Stop()
		}
	}
}

func (h *PolicyNodeWatcherTestHelper) ValidateActions(t *testing.T) {
	glog.Infof("Validating actions, %d callbacks recvd", len(h.TestEvents))
	if h.Watcher.ResourceVersion() != h.ResourceVersion() {
		t.Errorf("Got ResourceVersion %s, expected %s", h.Watcher.ResourceVersion(), h.ResourceVersion())
	}

	cbIdx := 0
	for opIdx, op := range h.testOperations {
		glog.Infof("Checking op %d, cb %d", opIdx, cbIdx)
		if op == nil {
			continue
		}

		switch op.Type {
		case watch.Error:
			if h.ErrValue == nil {
				t.Errorf("Should have encountered error")
			}
			return

		default:
			testEvent := h.TestEvents[cbIdx]
			if testEvent.eventType != fromWatcherType(op.Type) {
				t.Errorf("Action %d, Event %d should have had event type %s, got %s", opIdx, cbIdx, op.Type, testEvent.eventType)
			}
			cbIdx++
		}
	}

	if h.ErrValue != nil {
		t.Errorf("Encountered error %s", h.ErrValue)
	}
}

func NewPolicyNodeWatcherTestHelper() *PolicyNodeWatcherTestHelper {
	helper := &PolicyNodeWatcherTestHelper{
		FakeClient:      fake.NewSimpleClientset(),
		resourceVersion: 12345,
	}
	helper.FakeClient.PrependWatchReactor("*", helper.React)
	helper.Watcher = New(helper.FakeClient, helper.ResourceVersion())
	return helper
}

func TestPolicyNodeWatcher(t *testing.T) {
	helper := NewPolicyNodeWatcherTestHelper()

	helper.ActionClose()
	helper.ActionClose()
	helper.ActionAdd()
	helper.ActionClose()
	helper.ActionAdd()
	helper.ActionModify()
	helper.ActionDelete()
	helper.ActionClose()
	helper.ActionClose()
	helper.ActionAdd()

	helper.StopCheck = helper.DefaultStopper()

	watcher := helper.Watcher
	watcher.Run(helper)
	watcher.Wait()

	helper.ValidateActions(t)
}

func TestPolicyNodeWatcherError(t *testing.T) {
	helper := NewPolicyNodeWatcherTestHelper()

	helper.ActionClose()
	helper.ActionClose()
	helper.ActionAdd()
	helper.ActionClose()
	helper.ActionAdd()
	helper.ActionModify()
	helper.ActionDelete()
	helper.ActionClose()
	helper.ActionClose()
	helper.ActionError()

	helper.StopCheck = helper.DefaultStopper()

	watcher := helper.Watcher
	watcher.Run(helper)
	watcher.Wait()

	helper.ValidateActions(t)
}
