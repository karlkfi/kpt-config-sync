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
	"strconv"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"

	"github.com/golang/glog"

	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/client/policyhierarchy/fake"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	api_testing "k8s.io/client-go/testing"
)

const startResourceVersion = 12345

// testEvent is an event received during the testcase
type testEvent struct {
	eventType  EventType                      // The type of event we saw in the callback
	policyNode *policyhierarchy_v1.PolicyNode // The object from the callback.
}

// testOperation represents an event that the helper will simulate.
type testOperation struct {
	eventResourceVersion int64       // The resource version at which the event will be emitted
	close                bool        // If this should simualte the server closing the connection
	closeFired           bool        // If the close fired, this is to prevent infinite looping.
	event                watch.Event // The event emitted
}

type PolicyNodeWatcherTestHelper struct {
	FakeClient *fake.Clientset
	Watcher    *PolicyNodeWatcher

	// Records for OnEvent/OnError callbacks
	TestEvents []*testEvent // A list of events received from OnEvent
	ErrValue   error        // The error recieved from OnError
	CbCount    int          // The total number of OnEvent/OnError calls

	// The list of all operations that will be performed in the testcase.  A nil event represents
	// closing the event channel.
	testOperations   []testOperation
	testOperationIdx int

	// The StopCheck function, this callback will be invoked in each OnEvent/OnError call so that
	// the testcase can control when to stop watching.
	StopCheck func()
}

var _ EventHandler = &PolicyNodeWatcherTestHelper{}

func (h *PolicyNodeWatcherTestHelper) OnEvent(eventType EventType, policyNode *policyhierarchy_v1.PolicyNode) {
	h.CbCount++
	glog.Infof("Got OnEvent callback, cb count %d", h.CbCount)
	h.TestEvents = append(h.TestEvents, &testEvent{
		eventType:  eventType,
		policyNode: policyNode,
	})
	h.StopCheck()
}

func (h *PolicyNodeWatcherTestHelper) OnError(err error) {
	glog.Infof("Got OnError callback: %s", err)
	h.ErrValue = err
	h.CbCount++
	h.StopCheck()
}

func (h *PolicyNodeWatcherTestHelper) React(action api_testing.Action) (handled bool, ret watch.Interface, err error) {
	if action.GetResource().Group != policyhierarchy_v1.GroupName {
		return false, nil, nil
	}

	glog.Infof("In watch reactor for %s", spew.Sdump(action))
	fakeWatcher := watch.NewFake()
	resourceVersion, err := strconv.ParseInt(
		action.(api_testing.WatchActionImpl).WatchRestrictions.ResourceVersion, 10, 64)
	if err != nil {
		panic(errors.Wrapf(err, "Failed to parse resource version from action %#v", spew.Sdump(action)))
	}

	// Since we don't have a .find, we get the index of the element prior to where we should start
	// then add one so we don't have to check the last operation after the loop is over.
	h.testOperationIdx = -1
	for testOpIdx, testOperation := range h.testOperations {
		if resourceVersion < testOperation.eventResourceVersion {
			break
		}
		h.testOperationIdx = testOpIdx
	}
	h.testOperationIdx++

	go func() {
		for {
			glog.Infof("Operation idx %d", h.testOperationIdx)
			if len(h.testOperations) <= h.testOperationIdx {
				glog.Infof("No more operations, fake watcher stopping")
				fakeWatcher.Stop()
				return
			}

			op := &h.testOperations[h.testOperationIdx]
			h.testOperationIdx++
			if op.close {
				if !op.closeFired {
					glog.Infof("Fake watcher simulating CLOSE")
					op.closeFired = true
					fakeWatcher.Stop()
					return
				}
				glog.Infof("Skipping previously fired CLOSE")
				continue
			}

			glog.Infof("Fake watcher simulating %s", op.event.Type)
			fakeWatcher.Action(op.event.Type, op.event.Object)
		}
	}()

	return true, fakeWatcher, nil
}

// ActionAdd adds a simulated "Added" event to the test helper which will occur at
// the provided resource version
func (h *PolicyNodeWatcherTestHelper) ActionAdd(resourceVersion int64) {
	h.action(watch.Added, resourceVersion, resourceVersion)
}

// ActionModify adds a simulated "Modified" event to the test helper which will occur at
// the provided resource version
func (h *PolicyNodeWatcherTestHelper) ActionModify(resourceVersion int64) {
	h.action(watch.Modified, resourceVersion, resourceVersion)
}

// ActionDelete adds a simulated "Deleted" event to the test helper which will occur at
// the provided resource version, but show objectResourceVersion on the object's reesourceVersion field.
func (h *PolicyNodeWatcherTestHelper) ActionDelete(eventResourceVersion int64, objectResourceVersion int64) {
	h.action(watch.Deleted, eventResourceVersion, objectResourceVersion)
}

// ActionError adds a simulated "Error" event to the test helper which will occur at
// the provided resource version
func (h *PolicyNodeWatcherTestHelper) ActionError(resourceVersion int64) {
	h.queueEvent(&watch.Event{Type: watch.Error, Object: nil}, resourceVersion)
}

// ActionClose adds a simulated connection close to the test helper which will occur only once at
// the provided resource version
func (h *PolicyNodeWatcherTestHelper) ActionClose(resourceVersion int64) {
	h.testOperations = append(h.testOperations, testOperation{
		eventResourceVersion: resourceVersion,
		close:                true,
	})
}

func (h *PolicyNodeWatcherTestHelper) action(eventType watch.EventType, eventResourceVersion int64, objectResourceVersion int64) {
	h.queueEvent(
		&watch.Event{
			Type: eventType,
			Object: &policyhierarchy_v1.PolicyNode{
				ObjectMeta: meta_v1.ObjectMeta{
					ResourceVersion: fmt.Sprintf("%d", objectResourceVersion),
				},
			},
		},
		eventResourceVersion,
	)
}

func (h *PolicyNodeWatcherTestHelper) queueEvent(event *watch.Event, eventResourceVersion int64) {
	h.testOperations = append(h.testOperations, testOperation{
		eventResourceVersion: eventResourceVersion,
		event:                *event,
	})
}

// DefaultStopper sets up the default "stopper" function that will check for when to stop the listener.
func (h *PolicyNodeWatcherTestHelper) SetupDefaultStopper() {
	ops := 0
	hasError := false
	for _, testOperation := range h.testOperations {
		if !testOperation.close {
			ops++
			if testOperation.event.Type == watch.Error {
				hasError = true
			}
		}
	}
	glog.Infof("Creating default stopper for %d actions, error %t", ops, hasError)
	h.StopCheck = func() {
		glog.Infof("Stopper called, cb count %d, total ops %d", h.CbCount, ops)
		if h.CbCount >= ops && !hasError {
			h.Watcher.Stop()
		}
	}
}

// ValidateActions will check that the helper's callbacks saw all operations that were queued by
// the test case.
func (h *PolicyNodeWatcherTestHelper) ValidateActions(t *testing.T) {
	glog.Infof("Validating actions, %d callbacks recvd", len(h.TestEvents))

	// The last resource version is going to be the the resource version from the final op that is
	// neither close nor delete
	var lastResourceVersion int64
	for _, testOperation := range h.testOperations {
		if testOperation.close ||
			testOperation.event.Type == watch.Deleted ||
			testOperation.event.Type == watch.Error {
			continue
		}
		lastResourceVersion = testOperation.eventResourceVersion
	}

	if h.Watcher.ResourceVersion() != lastResourceVersion {
		t.Errorf("Got ResourceVersion %d, expected %d", h.Watcher.ResourceVersion(), lastResourceVersion)
	}

	cbIdx := 0
	for opIdx := range h.testOperations {
		testOperation := &h.testOperations[opIdx]
		glog.Infof("Checking op %d, cb %d", opIdx, cbIdx)
		if testOperation.close {
			continue
		}

		switch testOperation.event.Type {
		case watch.Error:
			if h.ErrValue == nil {
				t.Errorf("Should have encountered error")
			}
			return

		default:
			testEvent := h.TestEvents[cbIdx]
			if testEvent.eventType != fromWatcherType(testOperation.event.Type) {
				t.Errorf("Action %d, Event %d should have had event type %s, got %s",
					opIdx, cbIdx, testOperation.event.Type, testEvent.eventType)
			}
			cbIdx++
		}
	}

	if h.ErrValue != nil {
		t.Errorf("Encountered error %s", h.ErrValue)
	}
}

func (h *PolicyNodeWatcherTestHelper) RunAndCheckWatcher(t *testing.T) {
	glog.Info("---Starting test execution!---")
	h.SetupDefaultStopper()
	h.Watcher.Run(h)
	h.Watcher.Wait()
	h.ValidateActions(t)
}

// NewPolicyNodeWatcherTestHelper creates the test helper, this sets up hooks for the fake watch
// and creates the PolicyNodeWatcher
func NewPolicyNodeWatcherTestHelper(startResVersion int64) *PolicyNodeWatcherTestHelper {
	helper := &PolicyNodeWatcherTestHelper{
		FakeClient: fake.NewSimpleClientset(),
	}
	helper.FakeClient.PrependWatchReactor("*", helper.React)
	helper.Watcher = New(helper.FakeClient, startResVersion)
	return helper
}

func TestPolicyNodeWatcherBasic(t *testing.T) {
	startResVersion := int64(12345)
	helper := NewPolicyNodeWatcherTestHelper(startResVersion)

	helper.ActionAdd(startResVersion + 10)
	helper.ActionModify(startResVersion + 20)

	helper.RunAndCheckWatcher(t)
}

func TestPolicyNodeDeleteBeforeClose(t *testing.T) {
	startResVersion := int64(12345)
	helper := NewPolicyNodeWatcherTestHelper(startResVersion)

	helper.ActionAdd(startResVersion + 10)
	helper.ActionModify(startResVersion + 20)
	helper.ActionDelete(startResVersion+30, startResourceVersion+1)
	helper.ActionClose(startResVersion + 40)
	helper.ActionAdd(startResVersion + 50)

	helper.RunAndCheckWatcher(t)
}

func TestPolicyNodeDeleteAtEnd(t *testing.T) {
	startResVersion := int64(12345)
	helper := NewPolicyNodeWatcherTestHelper(startResVersion)

	helper.ActionAdd(startResVersion + 10)
	helper.ActionModify(startResVersion + 30)
	helper.ActionDelete(startResVersion+40, startResourceVersion+1)

	helper.RunAndCheckWatcher(t)
}

func TestPolicyNodeWatcher(t *testing.T) {
	startResVersion := int64(12345)
	helper := NewPolicyNodeWatcherTestHelper(startResVersion)

	helper.ActionClose(startResVersion + 10)
	helper.ActionClose(startResVersion + 20)
	helper.ActionAdd(startResVersion + 30)
	helper.ActionClose(startResVersion + 40)
	helper.ActionAdd(startResVersion + 50)
	helper.ActionModify(startResVersion + 60)
	helper.ActionDelete(startResVersion+70, startResVersion+1)
	helper.ActionClose(startResVersion + 80)
	helper.ActionClose(startResVersion + 90)
	helper.ActionAdd(startResVersion + 100)

	helper.RunAndCheckWatcher(t)
}

func TestPolicyNodeWatcherError(t *testing.T) {
	startResVersion := int64(12345)
	helper := NewPolicyNodeWatcherTestHelper(startResVersion)

	helper.ActionClose(startResVersion + 10)
	helper.ActionClose(startResVersion + 20)
	helper.ActionAdd(startResVersion + 30)
	helper.ActionClose(startResVersion + 40)
	helper.ActionAdd(startResVersion + 50)
	helper.ActionModify(startResVersion + 60)
	helper.ActionDelete(startResVersion+70, startResourceVersion+5)
	helper.ActionClose(startResVersion + 80)
	helper.ActionClose(startResVersion + 90)
	helper.ActionError(startResVersion + 100)

	helper.RunAndCheckWatcher(t)
}
