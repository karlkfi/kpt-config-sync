package watch

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

func fakeRunnable() Runnable {
	cfg := watcherConfig{
		reconciler: "test",
		startWatch: func(options metav1.ListOptions) (watch.Interface, error) {
			return watch.NewFake(), nil
		},
	}
	return NewFiltered(cfg)
}

func fakeError(gvk schema.GroupVersionKind) status.Error {
	return status.APIServerErrorf(errors.New("failed"), "watcher failed for %s", gvk.String())
}

func testRunnables(errOnType map[schema.GroupVersionKind]bool) func(context.Context, watcherConfig) (Runnable, status.Error) {
	return func(ctx context.Context, cfg watcherConfig) (runnable Runnable, err status.Error) {
		if errOnType[cfg.gvk] {
			return nil, fakeError(cfg.gvk)
		}
		return fakeRunnable(), nil
	}
}

func TestManager_Update(t *testing.T) {
	testCases := []struct {
		name string
		// watcherMap is the manager's map of watchers before the test begins.
		watcherMap map[schema.GroupVersionKind]Runnable
		// failedWatchers is the set of watchers which, if attempted to be
		// initialized, fail.
		failedWatchers map[schema.GroupVersionKind]bool
		// gvks is the map of gvks to update watches for. True/false indicates
		// validity to establish a new watch.
		gvks map[schema.GroupVersionKind]struct{}
		// wantWatchedTypes is the set of GVKS we want the Manager to be watching at
		// the end of the test.
		wantWatchedTypes []schema.GroupVersionKind
		// wantErr, if non-nil, reports that we want Update to return an error.
		wantErr error
	}{
		// Base Case.
		{
			name:           "no watchers and nothing declared",
			watcherMap:     map[schema.GroupVersionKind]Runnable{},
			failedWatchers: map[schema.GroupVersionKind]bool{},
			gvks:           map[schema.GroupVersionKind]struct{}{},
			wantErr:        nil,
		},
		// Watcher set mutations.
		{
			name:       "add watchers if declared",
			watcherMap: map[schema.GroupVersionKind]Runnable{},
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
				kinds.Role():      {},
			},
			wantWatchedTypes: []schema.GroupVersionKind{kinds.Namespace(), kinds.Role()},
		},
		{
			name: "keep watchers if still declared",
			watcherMap: map[schema.GroupVersionKind]Runnable{
				kinds.Namespace(): fakeRunnable(),
				kinds.Role():      fakeRunnable(),
			},
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
				kinds.Role():      {},
			},
			wantWatchedTypes: []schema.GroupVersionKind{kinds.Namespace(), kinds.Role()},
		},
		{
			name: "delete watchers if nothing declared",
			watcherMap: map[schema.GroupVersionKind]Runnable{
				kinds.Namespace(): fakeRunnable(),
				kinds.Role():      fakeRunnable(),
			},
			gvks: map[schema.GroupVersionKind]struct{}{},
		},
		{
			name: "add/keep/delete watchers",
			watcherMap: map[schema.GroupVersionKind]Runnable{
				kinds.Role():        fakeRunnable(),
				kinds.RoleBinding(): fakeRunnable(),
			},
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
				kinds.Role():      {},
			},
			wantWatchedTypes: []schema.GroupVersionKind{kinds.Namespace(), kinds.Role()},
		},
		// Error case.
		{
			name:       "error on starting watcher",
			watcherMap: map[schema.GroupVersionKind]Runnable{},
			failedWatchers: map[schema.GroupVersionKind]bool{
				kinds.Role(): true,
			},
			gvks: map[schema.GroupVersionKind]struct{}{
				kinds.Namespace(): {},
				kinds.Role():      {},
			},
			wantWatchedTypes: []schema.GroupVersionKind{kinds.Namespace()},
			wantErr:          fakeError(kinds.Role()),
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

			gotErr := m.UpdateWatches(context.Background(), tc.gvks)

			wantErr := status.Append(nil, tc.wantErr)
			if !errors.Is(wantErr, gotErr) {
				t.Errorf("got Update() error = %v, want %v", gotErr, tc.wantErr)
			}
			if diff := cmp.Diff(tc.wantWatchedTypes, m.watchedGVKs(), cmpopts.SortSlices(sortGVKs)); diff != "" {
				t.Error(diff)
			}

			gotNU := m.NeedsUpdate()
			wantNU := tc.wantErr != nil
			if gotNU != wantNU {
				t.Errorf("got NeedsUpdate() = %v, want %v", gotNU, wantNU)
			}
		})
	}
}

func sortGVKs(l, r schema.GroupVersionKind) bool {
	return l.String() < r.String()
}
