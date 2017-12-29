/*
Copyright 2017 The Stolos Authors.

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

import policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"

// EventHandler is an interface for handling events from the PolicyNodeWatcher.  Note that events
// will be called from a goroutine, but all callbacks are guaranteed to be serialized
// (both error and callback).
type EventHandler interface {
	// OnEvent is called when the watcher encounters an event.  This will pass both the
	// event type as well as the policy node on which the event operates.
	OnEvent(EventType, *policyhierarchy_v1.PolicyNode)

	// OnError is called when PolicyNodeWatcher encounters an error.  If this is
	// called PolicyNodeWatcher will tear itself down after the callback returns.
	OnError(err error)
}

type callbackEventHandler struct {
	eventCb func(EventType, *policyhierarchy_v1.PolicyNode)
	errorCb func(err error)
}

// NewEventHandler constructs an event handler from the provided callbacks. The callback eventCb will be called
// from OnEvent and the errorCb will be called from OnError.
func NewEventHandler(
	eventCb func(EventType, *policyhierarchy_v1.PolicyNode),
	errorCb func(err error)) EventHandler {
	return &callbackEventHandler{
		eventCb: eventCb,
		errorCb: errorCb,
	}
}

var _ EventHandler = &callbackEventHandler{}

// OnEvent implements EventHandler
func (h *callbackEventHandler) OnEvent(eventType EventType, policyNode *policyhierarchy_v1.PolicyNode) {
	h.eventCb(eventType, policyNode)
}

// OnError implements EventHandler
func (h *callbackEventHandler) OnError(err error) {
	h.errorCb(err)
}
