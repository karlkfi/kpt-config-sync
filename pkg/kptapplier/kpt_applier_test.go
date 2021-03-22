package kptapplier

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/apply"
	applyerror "sigs.k8s.io/cli-utils/pkg/apply/error"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeApplier struct {
	initErr error
	events  []event.Event
}

func newFakeApplier(err error, events []event.Event) *fakeApplier {
	return &fakeApplier{
		initErr: err,
		events:  events,
	}
}

func (a *fakeApplier) Run(_ context.Context, _ inventory.InventoryInfo, _ []*unstructured.Unstructured, _ apply.Options) <-chan event.Event {
	events := make(chan event.Event, len(a.events))
	go func() {
		for _, e := range a.events {
			events <- e
		}
		close(events)
	}()
	return events
}

func TestSync(t *testing.T) {
	resources, cache := prepareResources()

	testcases := []struct {
		name     string
		initErr  error
		events   []event.Event
		multiErr status.MultiError
		gvks     map[schema.GroupVersionKind]struct{}
	}{
		{
			name:     "applier init error",
			initErr:  errors.New("init error"),
			multiErr: ApplierError(errors.New("init error")),
		},
		{
			name: "unknown type for some resource",
			events: []event.Event{
				formApplyEvent(event.ApplyEventResourceUpdate, fakeID(), applyerror.NewUnknownTypeError(errors.New("unknown type"))),
				formApplyEvent(event.ApplyEventCompleted, nil, nil),
			},
			multiErr: ApplierError(errors.New("unknown type")),
			gvks:     map[schema.GroupVersionKind]struct{}{kinds.Deployment(): {}},
		},
		{
			name: "conflict error for some resource",
			events: []event.Event{
				formApplyEvent(event.ApplyEventResourceUpdate, fakeID(), inventory.NewInventoryOverlapError(errors.New("conflict"))),
				formApplyEvent(event.ApplyEventCompleted, nil, nil),
			},
			multiErr: ManagementConflictError(resources[0]),
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Deployment(): {},
				fakeKind():         {},
			},
		},
		{
			name: "failed to apply",
			events: []event.Event{
				formApplyEvent(event.ApplyEventResourceUpdate, fakeID(), applyerror.NewApplyRunError(errors.New("failed apply"))),
				formApplyEvent(event.ApplyEventCompleted, nil, nil),
			},
			multiErr: ApplierError(errors.New("failed apply")),
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Deployment(): {},
				fakeKind():         {},
			},
		},
		{
			name: "failed to prune",
			events: []event.Event{
				formPruneEvent(event.PruneEventFailed, event.Pruned, fakeID(), errors.New("failed pruning")),
				formPruneEvent(event.PruneEventCompleted, event.Pruned, nil, nil),
			},
			multiErr: ApplierError(errors.New("failed pruning")),
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Deployment(): {},
				fakeKind():         {},
			},
		},
		{
			name: "skipped pruning",
			events: []event.Event{
				formPruneEvent(event.PruneEventResourceUpdate, event.Pruned, fakeID(), nil),
				formPruneEvent(event.PruneEventResourceUpdate, event.PruneSkipped, deploymentID(), nil),
				formPruneEvent(event.PruneEventCompleted, event.Pruned, nil, nil),
			},
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Deployment(): {},
				fakeKind():         {},
			},
		},
		{
			name: "all passed",
			events: []event.Event{
				formApplyEvent(event.ApplyEventResourceUpdate, fakeID(), nil),
				formApplyEvent(event.ApplyEventResourceUpdate, deploymentID(), nil),
				formApplyEvent(event.ApplyEventCompleted, nil, nil),
				formPruneEvent(event.PruneEventCompleted, event.Pruned, nil, nil),
			},
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Deployment(): {},
				fakeKind():         {},
			},
		},
		{
			name: "all failed",
			events: []event.Event{
				formApplyEvent(event.ApplyEventResourceUpdate, fakeID(), applyerror.NewUnknownTypeError(errors.New("unknown type"))),
				formApplyEvent(event.ApplyEventResourceUpdate, deploymentID(), applyerror.NewApplyRunError(errors.New("failed apply"))),
				formApplyEvent(event.ApplyEventCompleted, nil, nil),
				formPruneEvent(event.PruneEventCompleted, event.Pruned, nil, nil),
			},
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Deployment(): {},
			},
			multiErr: status.Append(ApplierError(errors.New("failed pruning")), ApplierError(errors.New("failed apply"))),
		},
	}

	for _, tc := range testcases {
		applierFunc := func(c client.Client) (*clientSet, error) {
			return &clientSet{
				kptApplier: newFakeApplier(tc.initErr, tc.events),
			}, tc.initErr
		}
		applier := NewNamespaceApplier(nil, "test-namespace")
		applier.clientSetFunc = applierFunc
		gvks, errs := applier.sync(context.Background(), resources, cache)
		if diff := cmp.Diff(tc.gvks, gvks, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("%s: Diff of GVK map from Apply(): %s", tc.name, diff)
		}
		if tc.multiErr == nil {
			if errs != nil {
				t.Errorf("%s: unexpected error %v", tc.name, errs)
			}
		} else if errs == nil {
			t.Errorf("%s: expected some error, but not happened", tc.name)
		} else {
			actualErrs := errs.Errors()
			expectedErrs := tc.multiErr.Errors()
			if len(actualErrs) != len(expectedErrs) {
				t.Errorf("%s: number of error is not as expected %v", tc.name, actualErrs)
			} else {
				for i, actual := range actualErrs {
					expected := expectedErrs[i]
					if actual.Error() != expected.Error() && reflect.TypeOf(expected) != reflect.TypeOf(actual) {
						t.Errorf("%s: expected error %v but got %v", tc.name, expected, actual)
					}
				}
			}
		}
	}
}

func prepareResources() ([]client.Object, map[core.ID]client.Object) {
	u1 := fake.UnstructuredObject(kinds.Deployment(), core.Namespace("test-namespace"), core.Name("random-name"))
	u2 := fake.UnstructuredObject(schema.GroupVersionKind{
		Group:   "configsync.test",
		Version: "v1",
		Kind:    "Test",
	}, core.Namespace("test-namespace"), core.Name("random-name"))
	objs := []client.Object{u1, u2}
	cache := map[core.ID]client.Object{}
	for _, u := range objs {
		cache[core.IDOf(u)] = u
	}
	return objs, cache
}

func fakeID() *object.ObjMetadata {
	return &object.ObjMetadata{
		Namespace: "test-namespace",
		Name:      "random-name",
		GroupKind: schema.GroupKind{
			Group: "configsync.test",
			Kind:  "Test",
		},
	}
}

func deploymentID() *object.ObjMetadata {
	return &object.ObjMetadata{
		Namespace: "test-namespace",
		Name:      "random-name",
		GroupKind: kinds.Deployment().GroupKind(),
	}
}

func fakeKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "configsync.test",
		Version: "v1",
		Kind:    "Test",
	}
}

func formApplyEvent(t event.ApplyEventType, id *object.ObjMetadata, err error) event.Event {
	e := event.Event{
		Type: event.ApplyType,
		ApplyEvent: event.ApplyEvent{
			Type:  t,
			Error: err,
		},
	}
	if id != nil {
		e.ApplyEvent.Identifier = *id
	}
	return e
}

func formPruneEvent(t event.PruneEventType, op event.PruneEventOperation, id *object.ObjMetadata, err error) event.Event {
	e := event.Event{
		Type: event.PruneType,
		PruneEvent: event.PruneEvent{
			Type:      t,
			Error:     err,
			Operation: op,
		},
	}
	if id != nil {
		e.PruneEvent.Identifier = *id
	}
	return e
}
