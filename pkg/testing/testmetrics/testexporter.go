package testmetrics

import (
	"fmt"
	"sort"

	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/stats/view"
)

// TestExporter keeps exported metric view data in memory to aid in testing.
type TestExporter struct {
	rows []*view.Row
}

// RowSort implements sort.Interface based on the string representation of Row.
type RowSort []*view.Row

// ExportView records the view data.
func (e *TestExporter) ExportView(data *view.Data) {
	e.rows = data.Rows
}

// ValidateMetrics compares the exported view data with the expected metric data.
func (e *TestExporter) ValidateMetrics(v *view.View, want []*view.Row) string {
	view.Unregister(v)
	// Need to sort first because the exported row order is non-deterministic
	sort.Sort(RowSort(e.rows))
	sort.Sort(RowSort(want))
	return cmp.Diff(e.rows, want)
}

// RegisterMetrics collects data for the given views and reports data to the TestExporter.
func RegisterMetrics(v *view.View) *TestExporter {
	_ = view.Register(v)
	var e TestExporter
	view.RegisterExporter(&e)
	return &e
}

func (rs RowSort) Len() int           { return len(rs) }
func (rs RowSort) Swap(i, j int)      { rs[i], rs[j] = rs[j], rs[i] }
func (rs RowSort) Less(i, j int) bool { return fmt.Sprintf("%v", rs[i]) < fmt.Sprintf("%v", rs[j]) }
