package controller

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const canceled = "context canceled"

// CancelFilteringRecorder filters skips recording uninteresting events.  Wrapping a
// recorder into a filter ensures that all recorder call sites behave exactly
// the same.
type CancelFilteringRecorder struct {
	record.EventRecorder
}

var _ record.EventRecorder = (*CancelFilteringRecorder)(nil)

// Event implements record.EventRecorder.
func (r *CancelFilteringRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	if strings.Contains(message, canceled) || strings.Contains(reason, canceled) {
		// Context cancelations are expected, frequent and not actionable.
		return
	}
	r.EventRecorder.Event(object, eventtype, reason, message)
}
