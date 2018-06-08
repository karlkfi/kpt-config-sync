/*
Copyright 2017 The Nomos Authors.
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

// Package gcp provides functionality to receive a stream of policies from GCP via Kubernetes Policy API
// and converting them to Nomos Custom Resource Definition objects.
package gcp

import (
	"context"
	"io"
	"path"

	"github.com/gogo/googleapis/google/rpc"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/gogo/status"
	"github.com/golang/glog"
	watcher "github.com/google/nomos/clientgen/watcher/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	client_action "github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/policyimporter/actions"
	"github.com/google/nomos/pkg/util/policynode/validator"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
)

const (
	rootResource          = ""
	policyNodeResource    = "PolicyNode"
	clusterPolicyResource = "ClusterPolicy"
)

type resourceType string

// nolint: deadcode
// newActionGenerator returns a new actionGenerator.
// Only generate method is meant to to be called by users.
func newActionGenerator(stream watcher.Watcher_WatchClient, out chan<- actionVal, currentPolicies v1.AllPolicies, factories actions.Factories) *actionGenerator {
	return &actionGenerator{
		stream:          stream,
		out:             out,
		currentPolicies: currentPolicies,
		actionFactories: factories,
		gcpToK8SName:    make(map[string]string),
	}
}

type actionGenerator struct {
	// Stream client for Kubernetes Policy Watcher API.
	stream watcher.Watcher_WatchClient
	// Actions output channel.
	out chan<- actionVal
	// Current policies queried from K8S API.
	currentPolicies v1.AllPolicies
	actionFactories actions.Factories
	// Maps GCP resource name (e.g. folders/456, namespaces/789) to K8S name (e.g folders-456, backend)
	gcpToK8SName map[string]string
}

// generate watches changes from the grpc stream and generates corresponding K8S actions and
// writes them to the output channel.
//
// generate blocks until one of the events happens:
// 1. stream.Recv() returns an error
// 2. context.Done() returns true
// 3. An error occurs while processing a change
//
// In absent of these events, writing to output blocks indefinitely until the value is read by the receiver.
// If an error value is written to output channel, it's guaranteed to be the last written value.
// Output channel is closed when this method returns.
func (g *actionGenerator) generate(ctx context.Context) {
	defer close(g.out)

	initialState := true
	initialResources := make(map[string]*watcher.Change)

	for {
		changeBatch, err := g.stream.Recv()

		if err != nil {
			if err == io.EOF {
				glog.Info("Received graceful EOF")
				return
			}
			if s := status.Convert(err); s.Code() == codes.Canceled {
				glog.Info("Receive context cancelled")
				return
			}
			g.sendErr(ctx, errors.Wrapf(err, "failure on streaming receive"))
			return
		}

		for _, change := range changeBatch.Changes {
			glog.V(3).Infof("Received change: %#v", change)

			switch change.State {
			case watcher.Change_ERROR:
				g.sendErr(ctx, errors.Wrapf(unmarshalError(change), "error state for resource %q", change.Element))
				return
			case watcher.Change_INITIAL_STATE_SKIPPED:
				g.sendErr(ctx, errors.Errorf("unexpected state for resource %q: %s", change.Element, change.State))
				return
			}

			if initialState {
				initialResources[change.Element] = change
				if !change.Continued {
					glog.V(2).Infof("Processing initial state for %d resources", len(initialResources))
					actions, err := g.processInitialState(initialResources)
					if err != nil {
						g.sendErr(ctx, errors.Wrapf(err, "failed in processing initial state"))
						return
					}
					glog.V(2).Infof("Initial state generated %d actions", len(actions))
					for _, a := range actions {
						if !g.sendAction(ctx, a) {
							return
						}
					}
					// Received all changes in initial state
					initialState = false
					initialResources = nil
				}
				continue
			}

			glog.V(2).Infof("Processing incremental change")
			action, err := g.processIncrementalChange(change)
			if err != nil {
				g.sendErr(ctx, errors.Wrapf(err, "failed in processing incremental change"))
				return
			}
			glog.V(2).Infof("Incremental change generated action: %t", action != nil)
			if action != nil {
				if !g.sendAction(ctx, action) {
					return
				}
			}
		}
	}
}

// processInitialState generates the sequence of operations needed to transition from current to desired state of policies.
// Desired state of policies is parsed from initial Watcher changes and validated to ensure they form a valid hierarchy.
func (g *actionGenerator) processInitialState(resources map[string]*watcher.Change) ([]client_action.Interface, error) {
	if resources[""] == nil {
		return nil, errors.New("no initial state received for element \"\"")
	}

	var policies v1.AllPolicies
	policies.PolicyNodes = make(map[string]v1.PolicyNode)

	for name, change := range resources {
		glog.V(2).Infof("Resource %q has state: %s", name, change.State)
		switch change.State {
		case watcher.Change_EXISTS:
			t, err := policyResourceType(change.Element)
			if err != nil {
				return nil, err
			}
			switch t {
			case policyNodeResource:
				pn, err := unmarshalPolicyNode(change)
				if err != nil {
					return nil, err
				}
				policies.PolicyNodes[pn.Name] = *pn
				g.gcpToK8SName[change.Element] = pn.Name
			case clusterPolicyResource:
				cp, err := unmarshalClusterPolicy(change)
				if err != nil {
					return nil, err
				}
				policies.ClusterPolicy = cp
				// Ignore server-provider name of ClusterPolicy.
				policies.ClusterPolicy.Name = v1.ClusterPolicyName
			case rootResource:
				// Root element contains no policy
			}
		case watcher.Change_DOES_NOT_EXIST:
			glog.V(2).Infof("Ignoring resource %q", name)
		default:
			panic(errors.Errorf("unknown resource state: %s", change.State))
		}
	}
	v := validator.FromMap(policies.PolicyNodes)
	if err := v.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid PolicyNode hierarchy")
	}

	return actions.NewDiffer(g.actionFactories).Diff(g.currentPolicies, policies), nil
}

// processIncrementalChange converts the given Watcher change to a K8S API operation.
// It assumes the given state transition is valid as guaranteed by the API, if this
// is not the case, the operation will fail since PolicyNodeAdmissionController will reject it.
func (g *actionGenerator) processIncrementalChange(change *watcher.Change) (client_action.Interface, error) {
	gcpName := change.Element

	t, err := policyResourceType(change.Element)
	if err != nil {
		return nil, err
	}

	switch change.State {
	case watcher.Change_EXISTS:
		switch t {
		case policyNodeResource:
			pn, err := unmarshalPolicyNode(change)
			if err != nil {
				return nil, err
			}
			g.gcpToK8SName[gcpName] = pn.Name
			return g.actionFactories.PolicyNodeAction.NewUpsert(pn), nil
		case clusterPolicyResource:
			cp, err := unmarshalClusterPolicy(change)
			if err != nil {
				return nil, err
			}
			return g.actionFactories.ClusterPolicyAction.NewUpsert(cp), nil
		case rootResource:
			// Root element contains no policy
		}
	case watcher.Change_DOES_NOT_EXIST:
		switch t {
		case policyNodeResource:
			nodeName, ok := g.gcpToK8SName[gcpName]
			if !ok {
				return nil, errors.Errorf("cannot delete a non-existing resource %q", gcpName)
			}
			return g.actionFactories.PolicyNodeAction.NewDelete(nodeName), nil
		case clusterPolicyResource:
			return g.actionFactories.ClusterPolicyAction.NewDelete(v1.ClusterPolicyName), nil
		case rootResource:
			// Root element contains no policy
		}
	default:
		panic(errors.Errorf("unknown resource state: %s", change.State))
	}

	return nil, nil
}

func (g *actionGenerator) sendAction(ctx context.Context, a client_action.Interface) bool {
	return g.send(ctx, actionVal{action: a})
}

func (g *actionGenerator) sendErr(ctx context.Context, err error) bool {
	return g.send(ctx, actionVal{err: err})
}

func (g *actionGenerator) send(ctx context.Context, v actionVal) bool {
	select {
	case g.out <- v:
		return true
	case <-ctx.Done():
		glog.Warningf("Context is done: %v", ctx.Err())
		return false
	}
}

func unmarshalClusterPolicy(change *watcher.Change) (*v1.ClusterPolicy, error) {
	if change.State != watcher.Change_EXISTS {
		panic(errors.Errorf("must be in exist state, instead got: %s", change.State))
	}
	cp := &v1.ClusterPolicy{}
	err := ptypes.UnmarshalAny(change.Data, cp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal resource %q to ClusterPolicy", change.Element)
	}
	return cp, nil
}

func unmarshalPolicyNode(change *watcher.Change) (*v1.PolicyNode, error) {
	if change.State != watcher.Change_EXISTS {
		panic(errors.Errorf("must be in exist state, instead got: %s", change.State))
	}

	pn := &v1.PolicyNode{}
	err := ptypes.UnmarshalAny(change.Data, pn)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal resource %q to PolicyNode", change.Element)
	}
	return pn, nil
}

func unmarshalError(change *watcher.Change) error {
	if change.State != watcher.Change_ERROR {
		panic("Must be in error state")
	}

	s := &rpc.Status{}
	err := ptypes.UnmarshalAny(change.Data, s)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal error status")
	}
	p := status.ErrorProto(s)
	glog.Infof("%#v", p)
	return p
}

func policyResourceType(n string) (resourceType, error) {
	if n == "" {
		return rootResource, nil
	}
	if mustMatch("ClusterPolicy", n) {
		return clusterPolicyResource, nil
	}
	if mustMatch("PolicyNode", n) ||
		mustMatch("folders/*/PolicyNode", n) ||
		mustMatch("namespaces/*/PolicyNode", n) {
		return policyNodeResource, nil
	}
	return "", errors.Errorf("invalid resource name: %q", n)
}

func mustMatch(pattern, name string) bool {
	if m, err := path.Match(pattern, name); err != nil {
		panic(err)
	} else {
		return m
	}
}

type actionVal struct {
	action client_action.Interface
	err    error
}
