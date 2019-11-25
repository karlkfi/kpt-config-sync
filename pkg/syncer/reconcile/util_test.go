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

func TestFilterContextCancelled(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "error with no cause",
			err:  fmt.Errorf("simple"),
			want: fmt.Errorf("simple"),
		},
		{
			name: "error with cancelled cause",
			err:  errors.Wrap(context.Canceled, "outer error"),
		},
		{
			name: "error with other cause",
			err:  errors.Wrap(errors.New("some cause"), "outer error"),
			want: errors.Wrap(errors.New("some cause"), "outer error"),
		},
		{
			name: "multiple errors with context cancelled",
			err:  status.Append(nil, errors.Wrap(context.Canceled, "outer error"), errors.Wrap(context.Canceled, "another error")),
		},

		{
			name: "one error with context cancelled, two not",
			err:  status.Append(nil, errors.Wrap(context.Canceled, "outer error"), errors.Wrap(errors.New("some cause"), "another error"), errors.New("some error")),
			want: status.Append(nil, errors.Wrap(errors.New("some cause"), "another error"), errors.New("some error")),
		},
		{
			name: "filter nested multi error",
			err: status.Append(nil,
				fmt.Errorf("no cause"),
				status.Append(nil, errors.Wrap(context.Canceled, "outer error"), errors.Wrap(context.Canceled, "another error")),
			),
			want: status.Append(nil, fmt.Errorf("no cause")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterContextCancelled(tc.err)

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
