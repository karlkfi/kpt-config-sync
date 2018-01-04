/*
Copyright 2017 The Kubernetes Authors.
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

package policy_node_controller

import (
	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	listers_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/policyhierarchy/v1"
	"github.com/google/stolos/pkg/syncer"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/google/stolos/pkg/util/set/stringset"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/workqueue"
)

// PolicyNodeCopier is an implementation of PolicyNodeSyncerInterface that handles syncing
// PolicyNodes from a remote cluster.
type PolicyNodeCopier struct {
	localNodeLister    listers_v1.PolicyNodeLister
	localNodeInterface typed_v1.PolicyNodeInterface
	queue              workqueue.RateLimitingInterface
}

var _ syncer.PolicyNodeSyncerInterface = &PolicyNodeCopier{}

// NewPolicyNodeCopier creates a new policy node copier.
func NewPolicyNodeCopier(
	localNodeLister listers_v1.PolicyNodeLister,
	localNodeInterface typed_v1.PolicyNodeInterface,
	queue workqueue.RateLimitingInterface) *PolicyNodeCopier {
	return &PolicyNodeCopier{
		localNodeLister:    localNodeLister,
		localNodeInterface: localNodeInterface,
		queue:              queue,
	}
}

// OnCreate implements PolicyNodeSyncerInterface
func (p *PolicyNodeCopier) OnCreate(node *policyhierarchy_v1.PolicyNode) error {
	return p.onUpsert(node)
}

// OnCreate implements PolicyNodeSyncerInterface
func (p *PolicyNodeCopier) OnUpdate(old *policyhierarchy_v1.PolicyNode, new *policyhierarchy_v1.PolicyNode) error {
	return p.onUpsert(new)
}

// OnCreate implements PolicyNodeSyncerInterface
func (p *PolicyNodeCopier) onUpsert(node *policyhierarchy_v1.PolicyNode) error {
	p.queue.Add(NewPolicyNodeUpsertAction(node, p.localNodeLister, p.localNodeInterface))
	return nil
}

// OnDelete implements PolicyNodeSyncerInterface
func (p *PolicyNodeCopier) OnDelete(node *policyhierarchy_v1.PolicyNode) error {
	p.queue.Add(NewPolicyNodeDeleteAction(node, p.localNodeLister, p.localNodeInterface))
	return nil
}

// PeriodicResync implements PolicyNodeSyncerInterface
func (p *PolicyNodeCopier) PeriodicResync(nodes []*policyhierarchy_v1.PolicyNode) error {
	localNodes, err := p.localNodeLister.List(labels.Everything())
	if err != nil {
		return err
	}

	actions := p.computeActions(localNodes, nodes)
	for _, action := range actions {
		p.queue.Add(action)
	}
	return nil
}

// computeActions determines which policynode to delete during the resync. Creates will be handled
// by OnUpdate since every resource is "updated" during the resync. Deletes are handled by OnDelete
// but if we miss a delete due to being off, crashed, etc, this will garbage collect ones that
// we missed.
func (p *PolicyNodeCopier) computeActions(
	localNodes []*policyhierarchy_v1.PolicyNode,
	remoteNodes []*policyhierarchy_v1.PolicyNode) []actions.Interface {

	localNames := stringset.New()
	localNamesToNodes := make(map[string]*policyhierarchy_v1.PolicyNode)
	for _, n := range localNodes {
		localNames.Add(n.Name)
		localNamesToNodes[n.Name] = n
	}

	remoteNames := stringset.New()
	for _, n2 := range remoteNodes {
		remoteNames.Add(n2.Name)
	}

	needsDelete := localNames.Difference(remoteNames)

	actions := []actions.Interface{}
	needsDelete.ForEach(func(n string) {
		glog.Infof("Adding delete operation for %q", n)
		actions = append(actions, NewPolicyNodeDeleteAction(localNamesToNodes[n], p.localNodeLister, p.localNodeInterface))
	})

	return actions
}
