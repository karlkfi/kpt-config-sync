package controller

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const canceled = "context canceled"

// cancelFilteringRecorder filters skips recording uninteresting events.  Wrapping a
// recorder into a filter ensures that all recorder call sites behave exactly
// the same.
type cancelFilteringRecorder struct {
	// underlying is the EventRecorder we're filtering events to.
	// We keep the EventRecorder as an unexported member to ensure we don't
	// accidentally skip filtering, or else we can panic on context cancellation.
	underlying record.EventRecorder
}

var _ record.EventRecorder = &cancelFilteringRecorder{}

// Event implements record.EventRecorder.
func (r *cancelFilteringRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	if strings.Contains(message, canceled) || strings.Contains(reason, canceled) {
		// Context cancellations are expected, frequent and not actionable.
		return
	}
	r.underlying.Event(object, eventtype, reason, message)
}

// Eventf implements record.EventRecorder.
func (r *cancelFilteringRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	if strings.Contains(fmt.Sprintf(messageFmt, args...), canceled) || strings.Contains(reason, canceled) {
		// Context cancellations are expected, frequent and not actionable.
		return
	}
	r.underlying.Eventf(object, eventtype, reason, messageFmt, args)
}

// AnnotatedEventf implements record.EventRecorder.
func (r *cancelFilteringRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	if strings.Contains(fmt.Sprintf(messageFmt, args...), canceled) || strings.Contains(reason, canceled) {
		// Context cancellations are expected, frequent and not actionable.
		return
	}
	r.underlying.AnnotatedEventf(object, annotations, eventtype, reason, messageFmt, args)
}
