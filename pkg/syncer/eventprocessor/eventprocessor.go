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

// Package eventprocessor handles translating PolicyNode events into events for affected nodes in
// the subtree rooted at the given PolicyNode.
package eventprocessor

import (
	"github.com/golang/glog"
	policyhierarchy_informer_v1 "github.com/google/nomos/pkg/client/informers/externalversions/policyhierarchy/v1"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Factory returns a types.HandleFnProvider that will create a PolicyNodeEventProcessor with the
// passed informer.
func Factory(informer policyhierarchy_informer_v1.PolicyNodeInformer) types.HandleFnProvider {
	return func(queue workqueue.RateLimitingInterface) cache.ResourceEventHandlerFuncs {
		processor := &PolicyNodeEventProcessor{
			queue:    queue,
			informer: informer.Informer(),
		}
		return processor.HandlerFuncs()
	}
}

// PolicyNodeEventProcessor handles translating events for policy nodes into events for all spaces
// associated with the node in the hierarchy.
type PolicyNodeEventProcessor struct {
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer
}

// HandlerFuncs adapts PolicyNodeEventProcessor to a cache.ResourceEventHandlerFuncs
func (p *PolicyNodeEventProcessor) HandlerFuncs() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    p.OnAdd,
		UpdateFunc: p.OnUpdate,
		DeleteFunc: p.OnDelete,
	}
}

// OnAdd implements cache.ResourceEventHandler.
func (p *PolicyNodeEventProcessor) OnAdd(obj interface{}) {
	glog.Fatal("Not implemented")
}

// OnUpdate implements cache.ResourceEventHandler.
func (p *PolicyNodeEventProcessor) OnUpdate(oldObj, newObj interface{}) {
	glog.Fatal("Not implemented")
}

// OnDelete implements cache.ResourceEventHandler.
func (p *PolicyNodeEventProcessor) OnDelete(obj interface{}) {
	glog.Fatal("Not implemented")
}
