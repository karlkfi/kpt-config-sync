package reconcile

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
)

func TestFilterWithCause(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		filter error
		want   error
	}{
		{
			name: "error with no cause or filter",
			err:  fmt.Errorf("simple"),
			want: fmt.Errorf("simple"),
		},
		{
			name: "error with cause and no filter",
			err:  errors.Wrap(fmt.Errorf("cause"), "outer error"),
			want: errors.Wrap(fmt.Errorf("cause"), "outer error"),
		},
		{
			name:   "error with cause and filter",
			err:    errors.Wrap(context.Canceled, "outer error"),
			filter: context.Canceled,
		},
		{
			name: "multiple errors with cause and no filter",
			err:  status.From(errors.Wrap(context.Canceled, "outer error"), errors.Wrap(context.Canceled, "another error")),
			want: status.From(errors.Wrap(context.Canceled, "outer error"), errors.Wrap(context.Canceled, "another error")),
		},

		{
			name:   "multiple errors with cause, filter all",
			err:    status.From(errors.Wrap(context.Canceled, "outer error"), errors.Wrap(context.Canceled, "another error")),
			filter: context.Canceled,
		},
		{
			name:   "multiple errors with cause, filter some",
			err:    status.From(errors.Wrap(context.Canceled, "outer error"), fmt.Errorf("another error")),
			filter: context.Canceled,
			want:   status.From(fmt.Errorf("another error")),
		},
		{
			name: "filter nested multi error",
			err: status.From(
				fmt.Errorf("no cause"),
				status.From(errors.Wrap(context.Canceled, "outer error"), errors.Wrap(context.Canceled, "another error")),
			),
			filter: context.Canceled,
			want:   status.From(fmt.Errorf("no cause")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterWithCause(tc.err, tc.filter)

			if got == nil && tc.want == nil {
				return
			}
			if got == nil || tc.want == nil || got.Error() != tc.want.Error() {
				t.Errorf("filtered error is unexpected, got: %v\nwant: %v", got, tc.want)
			}
		})
	}
}

func TestPendingReconiclerRestart(t *testing.T) {
	testCases := []struct {
		name    string
		resGks  []schema.GroupKind
		toSync  []schema.GroupVersionKind
		wantRet []string
	}{
		{
			name:    "empty",
			resGks:  []schema.GroupKind{},
			toSync:  []schema.GroupVersionKind{},
			wantRet: nil,
		},
		{
			name: "ok - covered",
			resGks: []schema.GroupKind{
				{Kind: "ConfigMap"},
			},
			toSync: []schema.GroupVersionKind{
				{Kind: "ConfigMap", Version: "v1"},
				{Kind: "ConfigMap", Version: "v1beta1"},
			},
			wantRet: nil,
		},
		{
			name:   "ok - sync but no res",
			resGks: []schema.GroupKind{},
			toSync: []schema.GroupVersionKind{
				{Kind: "ConfigMap", Version: "v1"},
				{Kind: "ConfigMap", Version: "v1beta1"},
			},
			wantRet: nil,
		},
		{
			name: "need restart",
			resGks: []schema.GroupKind{
				{Kind: "ConfigMap"},
			},
			toSync:  []schema.GroupVersionKind{},
			wantRet: []string{"ConfigMap"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var resources []v1.GenericResources
			for _, gk := range tc.resGks {
				resources = append(resources, v1.GenericResources{
					Group: gk.Group,
					Kind:  gk.Kind,
				})
			}

			got := resourcesWithoutSync(resources, tc.toSync)
			if d := cmp.Diff(got, tc.wantRet); d != "" {
				t.Errorf("Wanted %v, got %v, diff: %s", tc.wantRet, got, d)
			}
		})
	}
}
