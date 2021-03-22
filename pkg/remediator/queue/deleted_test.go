package queue

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/testing/testmetrics"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWasDeleted(t *testing.T) {
	testCases := []struct {
		name string
		obj  client.Object
	}{
		{
			"object with no annotations",
			fake.ConfigMapObject(),
		},
		{
			"object with an annotation",
			fake.ConfigMapObject(core.Annotation("hello", "world")),
		},
		{
			"object with explicitly empty annotations",
			fake.ConfigMapObject(core.Annotations(map[string]string{})),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First verify that the object is not detected as deleted.
			ctx := context.Background()
			if WasDeleted(ctx, tc.obj) {
				t.Errorf("object was incorrectly detected as deleted: %v", tc.obj)
			}
			// Next mark the object as deleted and verify that it is now detected.
			deletedObj := MarkDeleted(ctx, tc.obj)
			if !WasDeleted(ctx, deletedObj) {
				t.Errorf("deleted object was not detected: %v", tc.obj)
			}
		})
	}
}

func TestDeleted_InternalErrorMetricValidation(t *testing.T) {
	m := testmetrics.RegisterMetrics(metrics.InternalErrorsView)
	ctx := context.Background()
	MarkDeleted(ctx, nil)
	wantMetrics := []*view.Row{
		{Data: &view.CountData{Value: 1}, Tags: []tag.Tag{{Key: metrics.KeyInternalErrorSource, Value: "remediator"}}},
	}
	if diff := m.ValidateMetrics(metrics.InternalErrorsView, wantMetrics); diff != "" {
		t.Errorf(diff)
	}
}
