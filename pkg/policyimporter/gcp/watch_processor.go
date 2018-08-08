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
	"io"
	"path"
	"time"

	"github.com/gogo/googleapis/google/rpc"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/gogo/status"
	"github.com/golang/glog"
	watcher "github.com/google/nomos/clientgen/watcher/v1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	client_action "github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/policyimporter"
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
type applicator func(client_action.Interface) error

// ToK8SNameMap maps GCP Watch element names to Kubernetes resource names that correspond to those names.
// Example: {"folders/456/PolicyNode": "folders-456"}
type ToK8SNameMap map[string]string

// Only the process() method is meant to to be called by users.
type watchProcessor struct {
	// Stream client for Kubernetes Policy Watcher API.
	stream watcher.Watcher_WatchClient
	// Actions application function.
	applyActionFn applicator
	// Current policies queried from K8S API.
	currentPolicies v1.AllPolicies
	actionFactories actions.Factories
	// Maps GCP resource name (e.g. folders/456, projects/789) to K8S name (e.g folders-456, backend)
	gcpToK8SName ToK8SNameMap
	// Whether initial state has been processed.
	initialStateDone bool
	// Function that cancels the watch streaming RPC context
	cancelWatchFn func()
	// How long we wait to receive a message before timing out
	timeout time.Duration
}

// process watches changes from the grpc stream and generates corresponding K8S actions and
// passes them to applyActionFn.
//
// process blocks until one of the events happens:
// 1. stream.Recv() returns an error
// 2. stream does not return a value by the time timeout expires
// 3. An error occurs while processing a change
func (p *watchProcessor) process() ([]byte, error) {
	resources := make(map[string]*watcher.Change)
	var resumeMarker []byte

	for {
		t := time.AfterFunc(p.timeout, func() {
			glog.Error("Liveliness timeout on watch process.")
			p.cancelWatchFn()
		})
		changeBatch, err := p.stream.Recv()
		t.Stop()

		if err != nil {
			if err == io.EOF {
				glog.Info("Received graceful EOF")
				return resumeMarker, nil
			}
			if s := status.Convert(err); s.Code() == codes.Canceled {
				glog.Info("Receive context cancelled")
				return resumeMarker, nil
			}
			return resumeMarker, errors.Wrapf(err, "failure on streaming receive")
		}

		for _, change := range changeBatch.Changes {
			glog.V(3).Infof("Received change: %#v", change)

			switch change.State {
			case watcher.Change_ERROR:
				return resumeMarker, errors.Wrapf(unmarshalError(change), "error state for resource %q", change.Element)
			case watcher.Change_INITIAL_STATE_SKIPPED:
				return resumeMarker, errors.Errorf("unexpected state for resource %q: %s", change.Element, change.State)
			}

			resources[change.Element] = change
			if !change.Continued {
				updatedPolicies, err := p.processAtomicGroup(resources)
				if err != nil {
					policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
					return resumeMarker, errors.Wrapf(err, "failed in processing atomic group")
				}
				a := actions.NewDiffer(p.actionFactories).Diff(p.currentPolicies, *updatedPolicies)
				glog.V(2).Infof("Processing of atomic group generated %d actions", len(a))
				p.currentPolicies = *updatedPolicies
				p.initialStateDone = true
				resources = make(map[string]*watcher.Change)
				policyimporter.Metrics.Nodes.Set(float64(len(p.currentPolicies.PolicyNodes)))
				for _, a := range a {
					if err := p.applyActionFn(a); err != nil {
						policyimporter.Metrics.PolicyStates.WithLabelValues("failed").Inc()
						return resumeMarker, errors.Wrapf(err, "failed in applying action %#v", a)
					}
				}
				policyimporter.Metrics.PolicyStates.WithLabelValues("succeeded").Inc()
				if change.GetResumeMarker() != nil {
					resumeMarker = change.GetResumeMarker()
				}
			}
		}
	}
}

// processAtomicGroup returns updated state of policies given the changes.
// It checks that the updated policies form a valid hierarchy.
func (p *watchProcessor) processAtomicGroup(resources map[string]*watcher.Change) (*v1.AllPolicies, error) {
	if !p.initialStateDone && resources[""] == nil {
		return nil, errors.New("no initial state received for element \"\"")
	}

	var updatedPolicies v1.AllPolicies
	if p.initialStateDone {
		p.currentPolicies.DeepCopyInto(&updatedPolicies)
	} else {
		updatedPolicies.PolicyNodes = make(map[string]v1.PolicyNode)
	}

	for name, change := range resources {
		glog.V(2).Infof("%q %s", name, change.State)
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
				updatedPolicies.PolicyNodes[pn.Name] = *pn
				glog.V(2).Infof("%q -> nomos.dev/v1/PolicyNodes/%s", name, pn.Name)
				p.gcpToK8SName[name] = pn.Name
			case clusterPolicyResource:
				cp, err := unmarshalClusterPolicy(change)
				if err != nil {
					return nil, err
				}
				updatedPolicies.ClusterPolicy = cp
				// Ignore server-provided name of ClusterPolicy.
				updatedPolicies.ClusterPolicy.Name = v1.ClusterPolicyName
			case rootResource:
				// Root element contains no policy
			}

		case watcher.Change_DOES_NOT_EXIST:
			if !p.initialStateDone {
				glog.V(2).Infof("Initial state of resource cannot be %s", change.State)
				continue
			}
			switch t {
			case policyNodeResource:
				nodeName, ok := p.gcpToK8SName[name]
				if !ok {
					glog.Warningf("cannot delete non-existing resource %q", name)
					continue
				}
				glog.V(2).Infof("%q -> nomos.dev/v1/PolicyNodes/%s", name, nodeName)
				delete(updatedPolicies.PolicyNodes, nodeName)
			case clusterPolicyResource:
				updatedPolicies.ClusterPolicy = nil
			case rootResource:
				// Root element contains no policy
			}
		default:
			panic(errors.Errorf("unknown resource state: %s", change.State))
		}
	}

	glog.V(3).Infof("Update state of policies: %#v", updatedPolicies)

	v := validator.FromMap(updatedPolicies.PolicyNodes)
	if err := v.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid PolicyNode hierarchy")
	}
	return &updatedPolicies, nil
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
		mustMatch("projects/*/PolicyNode", n) {
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
