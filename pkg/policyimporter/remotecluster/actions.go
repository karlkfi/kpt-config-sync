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

package remotecluster

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	policyhierarchy_v1 "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	listers_v1 "github.com/google/stolos/pkg/client/listers/policyhierarchy/v1"
	typed_v1 "github.com/google/stolos/pkg/client/policyhierarchy/typed/policyhierarchy/v1"
	"github.com/google/stolos/pkg/syncer/actions"
	"github.com/pkg/errors"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type policyNodeAction struct {
	// PolicyNode name
	name string

	// Remote PolicyNode being synced
	remoteNode *policyhierarchy_v1.PolicyNode

	// Name of the operation being performed, mostly here for logging purposes.
	operation string

	// Local cluster API access related objects
	localNodeLister    listers_v1.PolicyNodeLister
	localNodeInterface typed_v1.PolicyNodeInterface
}

// Resource implements the action interface.
func (p *policyNodeAction) Resource() string {
	return "policynode"
}

// Operation implements the action interface.
func (p *policyNodeAction) Operation() string {
	return p.operation
}

// String implements the action interface.
func (p *policyNodeAction) String() string {
	return fmt.Sprintf("%s.%s.%s", p.Resource(), p.remoteNode.Name, p.Operation())
}

// Namespace implements the action interface.
func (p *policyNodeAction) Namespace() string {
	// PolicyNodes are cluster level resources.
	return ""
}

// PolicyNodeUpsertAction will create or update a policynode when executed
type PolicyNodeUpsertAction struct {
	policyNodeAction
}

var _ actions.Interface = &PolicyNodeUpsertAction{}

// NewPolicyNodeUpsertAction creates a new PolicyNodeUpsertAction
func NewPolicyNodeUpsertAction(
	policyNode *policyhierarchy_v1.PolicyNode,
	localPolicyNodeLister listers_v1.PolicyNodeLister,
	localPolicyNodeInterface typed_v1.PolicyNodeInterface) *PolicyNodeUpsertAction {
	return &PolicyNodeUpsertAction{
		policyNodeAction: policyNodeAction{
			name:               policyNode.Name,
			remoteNode:         policyNode,
			localNodeLister:    localPolicyNodeLister,
			localNodeInterface: localPolicyNodeInterface,
			operation:          "upsert",
		},
	}
}

// Execute implements the action interface.
func (p *PolicyNodeUpsertAction) Execute() error {
	localNode, err := p.localNodeLister.Get(p.name)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return p.create()
		}
		return errors.Wrapf(err, "Failed to get local policynode %q during upsert", p.name)
	}

	return p.update(localNode)
}

func (p *PolicyNodeUpsertAction) create() error {
	createdPolicyNode, err := p.localNodeInterface.Create(canonicalCopy(p.remoteNode))

	if err != nil {
		if api_errors.IsAlreadyExists(err) {
			glog.Infof("policynode %q already exists", p.name)
			return nil
		}
		return errors.Wrapf(err, "Failed to create policynode %q during upsert", p.name)
	}
	glog.Infof("Created policynode %q, resourceVersion %s", p.name, createdPolicyNode.ResourceVersion)
	return nil
}

func (p *PolicyNodeUpsertAction) update(localNode *policyhierarchy_v1.PolicyNode) error {
	if Equal(localNode, p.remoteNode) {
		glog.Infof("Existing policynode %q does not need to be updated", p.name)
		return nil
	}

	u := canonicalCopy(p.remoteNode)
	u.ResourceVersion = localNode.ResourceVersion
	u, err := p.localNodeInterface.Update(u)
	if err != nil {
		return errors.Wrapf(err, "Failed to update policynode %q", p.name)
	}

	glog.Infof("Updated policynode %q, resourceVersion %s", p.name, u.ResourceVersion)
	return nil
}

// PolicyNodeDeleteAction will delete a policynode when executed
type PolicyNodeDeleteAction struct {
	policyNodeAction
}

var _ actions.Interface = &PolicyNodeDeleteAction{}

// NewPolicyNodeDeleteAction creates a new PolicyNodeDeleteAction
func NewPolicyNodeDeleteAction(
	policyNode *policyhierarchy_v1.PolicyNode,
	localPolicyNodeLister listers_v1.PolicyNodeLister,
	localPolicyNodeInterface typed_v1.PolicyNodeInterface) *PolicyNodeDeleteAction {
	return &PolicyNodeDeleteAction{
		policyNodeAction: policyNodeAction{
			name:               policyNode.Name,
			remoteNode:         policyNode,
			localNodeLister:    localPolicyNodeLister,
			localNodeInterface: localPolicyNodeInterface,
			operation:          "delete",
		},
	}
}

// Execute implements PolicyNodeAction
func (p *PolicyNodeDeleteAction) Execute() error {
	_, err := p.localNodeLister.Get(p.name)
	if err != nil {
		if api_errors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "Failed to get policynode %q from cache", p.name)
	}

	err = p.localNodeInterface.Delete(p.name, &meta_v1.DeleteOptions{})
	if err != nil && !api_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete policynode %q", p.name)
	}
	glog.Infof("Deleted policynode %q", p.name)
	return nil
}

// Creates a canonical copy of remote cluster node by discarding fields that don't make sense in the
// local cluster.
// TODO(frankf): We might want to also copy certain annotations and labels in the future.
func canonicalCopy(remoteNode *policyhierarchy_v1.PolicyNode) *policyhierarchy_v1.PolicyNode {
	return &policyhierarchy_v1.PolicyNode{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: remoteNode.Name,
		},
		TypeMeta: remoteNode.TypeMeta,
		Spec:     remoteNode.Spec,
	}
}

func Equal(left *policyhierarchy_v1.PolicyNode, right *policyhierarchy_v1.PolicyNode) bool {
	return reflect.DeepEqual(canonicalCopy(left), canonicalCopy(right))
}
