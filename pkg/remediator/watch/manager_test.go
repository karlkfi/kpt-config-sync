package watch

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

func fakeRunnable() Runnable {
	return &filteredWatcher{
		base:      watch.NewFake(),
		resources: nil,
		queue:     nil,
	}
}

func testRunnables(errOnType map[schema.GroupVersionKind]bool) func(watcherOptions) (Runnable, status.Error) {
	return func(options watcherOptions) (runnable Runnable, err status.Error) {
		if errOnType[options.gvk] {
			return nil, FailedToStartWatcher(
				errors.Errorf("error creating watcher for %v", options.gvk),
			)
		}
		return fakeRunnable(), nil
	}
}

var ignoreRunnables = cmpopts.IgnoreInterfaces(struct{ Runnable }{})

func TestManager_Update(t *testing.T) {
	testCases := []struct {
		name string
		// watcherMap is the manager's map of watchers before the test begins.
		watcherMap map[schema.GroupVersionKind]Runnable
		// failedWatchers is the set of watchers which, if attempted to be
		// initialized, fail.
		failedWatchers map[schema.GroupVersionKind]bool
		// objects is the list of objects to update.
		objects []core.Object
		// wantWatchedTypes is the set of types we want the Manager to report it is
		// watching. This set is equivalent to the set of keys we expect the
		// watcherMap to have at the end of the test.
		wantWatchedTypes map[schema.GroupVersionKind]bool
		// wantErr, if non-nil, reports that we want Update to return an error.
		wantErr error
	}{
		// Base Case.
		{
			name:             "no watchers and nothing declared",
			watcherMap:       map[schema.GroupVersionKind]Runnable{},
			failedWatchers:   map[schema.GroupVersionKind]bool{},
			objects:          []core.Object{},
			wantWatchedTypes: map[schema.GroupVersionKind]bool{},
			wantErr:          nil,
		},
		// Watcher set mutations.
		{
			name:       "add watchers if declared",
			watcherMap: map[schema.GroupVersionKind]Runnable{},
			objects: []core.Object{
				fake.NamespaceObject("shipping"),
				fake.RoleObject(),
			},
			wantWatchedTypes: map[schema.GroupVersionKind]bool{
				kinds.Namespace(): true,
				kinds.Role():      true,
			},
		},
		{
			name: "keep watchers if still declared",
			watcherMap: map[schema.GroupVersionKind]Runnable{
				kinds.Namespace(): fakeRunnable(),
				kinds.Role():      fakeRunnable(),
			},
			objects: []core.Object{
				fake.NamespaceObject("shipping"),
				fake.RoleObject(),
			},
			wantWatchedTypes: map[schema.GroupVersionKind]bool{
				kinds.Namespace(): true,
				kinds.Role():      true,
			},
		},
		{
			name: "delete watchers if nothing declared",
			watcherMap: map[schema.GroupVersionKind]Runnable{
				kinds.Namespace(): fakeRunnable(),
				kinds.Role():      fakeRunnable(),
			},
			objects: []core.Object{},
		},
		{
			name: "add/keep/delete watchers",
			watcherMap: map[schema.GroupVersionKind]Runnable{
				kinds.Role():        fakeRunnable(),
				kinds.RoleBinding(): fakeRunnable(),
			},
			objects: []core.Object{
				fake.NamespaceObject("shipping"),
				fake.RoleObject(),
			},
			wantWatchedTypes: map[schema.GroupVersionKind]bool{
				kinds.Namespace(): true,
				kinds.Role():      true,
			},
		},
		// Error case.
		{
			name:       "error on starting watcher",
			watcherMap: map[schema.GroupVersionKind]Runnable{},
			failedWatchers: map[schema.GroupVersionKind]bool{
				kinds.Role(): true,
			},
			objects: []core.Object{
				fake.RoleObject(),
			},
			wantWatchedTypes: map[schema.GroupVersionKind]bool{
				kinds.Role(): false,
			},
			wantErr: FailedToStartWatcher(errors.New("failed")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			options := &Options{
				watcherFunc: testRunnables(tc.failedWatchers),
			}
			m, err := NewManager(":test", nil, nil, &declared.Resources{}, options)
			if err != nil {
				t.Fatal(err)
			}

			watched, err := m.Update(tc.objects)

			wantErr := status.Append(nil, tc.wantErr)
			if !errors.Is(wantErr, err) {
				t.Errorf("got Update() error = %v, want %v", err, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantWatchedTypes, watched, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}

			wantWatchers := make(map[schema.GroupVersionKind]Runnable)
			for gvk, want := range tc.wantWatchedTypes {
				if want {
					wantWatchers[gvk] = nil
				}
			}
			if diff := cmp.Diff(wantWatchers, m.watcherMap, ignoreRunnables, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
