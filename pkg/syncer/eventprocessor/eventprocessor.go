/*
Copyright 2018 The Nomos Authors.
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
// Reviewed by sunilarora

// Package eventprocessor handles translating PolicyNode events into events for affected nodes in
// the subtree rooted at the given PolicyNode.
package eventprocessor

import (
	policyhierarchyinformer_v1 "github.com/google/nomos/clientgen/informers/policyhierarchy/policyhierarchy/v1"
	"github.com/google/nomos/pkg/syncer/hierarchy"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	policyhierarchy_v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
)

// Factory returns a types.HandleFnProvider that will create a PolicyNodeEventProcessor with the
// passed informer.
func Factory(informer policyhierarchyinformer_v1.PolicyNodeInformer) types.HandleFnProvider {
	return func(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
		processor := &PolicyNodeEventProcessor{
			queue:     queue,
			hierarchy: hierarchy.New(informer),
		}
		return processor
	}
}

// PolicyNodeEventProcessor handles translating events for policy nodes into events for all spaces
// associated with the node in the hierarchy.
type PolicyNodeEventProcessor struct {
	queue     workqueue.Interface
	hierarchy hierarchy.Interface
}

// OnAdd implements cache.ResourceEventHandler.
func (p *PolicyNodeEventProcessor) OnAdd(obj interface{}) {
	p.subtreeEvents(obj.(*policyhierarchy_v1.PolicyNode))
}

// OnUpdate implements cache.ResourceEventHandler.
func (p *PolicyNodeEventProcessor) OnUpdate(oldObj, newObj interface{}) {
	p.subtreeEvents(newObj.(*policyhierarchy_v1.PolicyNode))
}

// subtreeEvents handles generating all events for a subtree.
func (p *PolicyNodeEventProcessor) subtreeEvents(policyNode *policyhierarchy_v1.PolicyNode) {
	names, err := p.hierarchy.Subtree(policyNode.Name)
	if err != nil {
		if hierarchy.IsNotFoundError(err) {
			// This is possible if the resource is added/updated then deleted before we process this event.
			return
		}
		// The informer is only expected to error out if something is not set up correctly (not found
		// is the only error we should expect at this point).
		panic(errors.Wrapf(err, "encountered programmer error"))
	}
	for _, name := range names {
		p.queue.Add(name)
	}
}

// OnDelete implements cache.ResourceEventHandler.
func (p *PolicyNodeEventProcessor) OnDelete(obj interface{}) {
	policyNode := obj.(*policyhierarchy_v1.PolicyNode)
	p.queue.Add(policyNode.Name)
}
