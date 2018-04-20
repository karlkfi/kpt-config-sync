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

package gcp

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"
	any_proto "github.com/golang/protobuf/ptypes/any"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/actions"
	gcp_testing "github.com/google/nomos/pkg/policyimporter/gcp/testing"
	"github.com/google/nomos/pkg/util/policynode"
	"google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// Example PolicyNode resources
	orgPN, nsPN *v1.PolicyNode
	// Example any proto marshalled policies resources.
	emptyProto, orgPNProto, orgCPProto, folderPNProto, nsPNProto *any_proto.Any
)

var testCases []testCase

type testCase struct {
	testName string
	// 1st ChangeBatch returned by stream.Recv()
	batch1 []*watcher.Change
	// 2nd ChangeBatch returned by stream.Recv()
	batch2 []*watcher.Change
	// Current policies returned by K8S API server at start up.
	currentPolicies v1.AllPolicies
	// Ordered list of actions expected to be written to output channel.
	expectedActions []string
	// Whether an error is expected as the last value written to the output channel.
	expectedError bool
}

func init() {
	emptyProto = &any_proto.Any{}
	anError, err := ptypes.MarshalAny(status.New(codes.DeadlineExceeded, "deadline exceeded when receiving policies").Proto())
	if err != nil {
		panic(err)
	}
	orgPN = newPolicyNode("organization-123", "", true, nil)
	orgPNProto = toAnyProto(orgPN)
	orgCPProto = toAnyProto(newClusterPolicy("organization-123"))
	folderPNProto = toAnyProto(newPolicyNode("folder-456", "organization-123", true, nil))
	nsPN = newPolicyNode("backend", "folder-456", false, nil)
	nsPNProto = toAnyProto(nsPN)

	testCases = []testCase{
		{
			testName: "No change",
		},
		{
			testName: "Initial state org",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
			},
		},
		{
			testName: "Initial state org with error",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_ERROR, Continued: false, Data: anError},
			},
			expectedError: true,
		},
		{
			testName: "Initial state org with initial skipped",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_INITIAL_STATE_SKIPPED, Continued: false, Data: emptyProto},
			},
			expectedError: true,
		},
		{
			testName: "Initial state org with PolicyNode",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
			},
		},
		{
			testName: "Initial state org with ClusterPolicy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "ClusterPolicy", State: watcher.Change_EXISTS, Continued: false, Data: orgCPProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/upsert",
			},
		},
		{
			testName: "Initial state org with PolicyNode and ClusterPolicy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: orgPNProto},
				{Element: "ClusterPolicy", State: watcher.Change_EXISTS, Continued: false, Data: orgCPProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/upsert",
			},
		},
		{
			testName: "Initial state duplicates",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: orgPNProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
			},
		},
		{
			testName: "Initial state with non-existent resource",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
			},
		},
		{
			testName: "Initial state with valid hierarchy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: orgPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: folderPNProto},
				{Element: "namespaces/backend/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: nsPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/backend/upsert",
			},
		},
		{
			testName: "Initial state with existing policies",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: orgPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: folderPNProto},
				{Element: "namespaces/backend/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: nsPNProto},
			},
			currentPolicies: v1.AllPolicies{PolicyNodes: map[string]v1.PolicyNode{
				"organization-123": *orgPN,
			}},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/backend/upsert",
			},
		},
		{
			testName: "Initial state multi batch",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: orgPNProto},
			},
			batch2: []*watcher.Change{
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
			},
		},
		{
			testName: "Initial state changes not in tree invariant order",
			batch1: []*watcher.Change{
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: folderPNProto},
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
			},
		},
		{
			testName: "Initial state folder element",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "folders/456", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
			},
			expectedError: true,
		},
		{
			testName: "Initial state with invalid hierarchy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
			},
			expectedError: true,
		},
		// TODO(frankf): Fix incompatibily between golang/protobuf and gogo/protobuf.
		//{
		//	testName: "Initial state with unmarshal error",
		//	batch1: []*watcher.Change{
		//		{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
		//		{Element: "ClusterPolicy", State: watcher.Change_EXISTS, Continued: false, Data: orgCPProto},
		//	},
		//	expectedError: false,
		//},
		{
			testName: "Initial state no root",
			batch1: []*watcher.Change{
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			expectedError: true,
		},
		{
			testName: "Initial state non-org with ClusterPolicy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: orgPNProto},
				{Element: "folders/456/ClusterPolicy", State: watcher.Change_EXISTS, Continued: false, Data: orgCPProto},
			},
			expectedError: true,
		},
		{
			testName: "Incremental change",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/organization-123/delete",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/delete",
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
			},
		},
	}
}

func TestGen(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stream := gcp_testing.NewMockWatcher_WatchClient(ctrl)
			out := make(chan actionVal)
			// Factories take nil arguments since we don't need to apply the actions for these tests.
			g := newActionGenerator(
				stream, out, tc.currentPolicies, actions.NewFactories(nil, nil, nil))

			stream.EXPECT().Recv().Return(&watcher.ChangeBatch{tc.batch1}, nil)
			if tc.batch2 != nil {
				stream.EXPECT().Recv().Return(&watcher.ChangeBatch{tc.batch2}, nil)
			}
			if !tc.expectedError {
				stream.EXPECT().Recv().Return(nil, io.EOF)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
			defer cancel()

			go g.generate(ctx)

			var actualError bool
			var actualErrorMsg string
			var actualActions []string
			for v := range out {
				if v.err != nil {
					actualError = true
					actualErrorMsg = v.err.Error()
				} else {
					if actualError {
						t.Fatalf("Error must be last actionVal send to channel")
					}
					actualActions = append(actualActions, v.action.String())
				}
			}

			if diff := deep.Equal(actualError, tc.expectedError); diff != nil {
				t.Fatalf("Actual and expected error don't match: %v. Actual error: %s", diff, actualErrorMsg)
			}

			if diff := deep.Equal(actualActions, tc.expectedActions); diff != nil {
				t.Fatalf("Actual and expected actions don't match: %v", diff)
			}
		})
	}
}

// Test that generate() returns when context is done.
func TestDone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stream := gcp_testing.NewMockWatcher_WatchClient(ctrl)
	out := make(chan actionVal)
	g := newActionGenerator(stream, out, v1.AllPolicies{}, actions.NewFactories(nil, nil, nil))

	stream.EXPECT().Recv().Return(&watcher.ChangeBatch{[]*watcher.Change{
		{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
		{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
	},
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	g.generate(ctx)
}

func TestRecvErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stream := gcp_testing.NewMockWatcher_WatchClient(ctrl)
	out := make(chan actionVal)
	g := newActionGenerator(stream, out, v1.AllPolicies{}, actions.NewFactories(nil, nil, nil))

	stream.EXPECT().Recv().Return(nil, errors.New("receive error"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go g.generate(ctx)

	var vals []actionVal
	for v := range out {
		vals = append(vals, v)
	}
	expectedVals := []actionVal{{err: errors.New("receive error")}}
	if diff := deep.Equal(vals, expectedVals); diff != nil {
		t.Fatalf("Actual and expected actions don't match: %v", diff)
	}
}

func newPolicyNode(name string, parent string, policyspace bool, policies *v1.Policies) *v1.PolicyNode {
	pn := policynode.NewPolicyNode(name,
		&v1.PolicyNodeSpec{
			Policyspace: policyspace,
			Parent:      parent,
		})
	if policies != nil {
		pn.Spec.Policies = *policies
	}
	return pn
}

func newClusterPolicy(name string) *v1.ClusterPolicy {
	return policynode.NewClusterPolicy(name,
		&v1.ClusterPolicySpec{
			Policies: v1.ClusterPolicies{},
		})
}

func toAnyProto(m interface{}) *any_proto.Any {
	switch v := m.(type) {
	case *v1.PolicyNode:
		p, err := ptypes.MarshalAny(v)
		if err != nil {
			panic(p)
		}
		return p
	case *v1.ClusterPolicy:
		p, err := ptypes.MarshalAny(v)
		if err != nil {
			panic(p)
		}
		return p
	default:
		panic("Invalid type")
	}
}

func TestPolicyResourceType(t *testing.T) {
	tcs := []struct {
		input          string
		expectedOutput resourceType
		expectedError  bool
	}{
		{
			input:          "",
			expectedOutput: rootResource,
		},
		{
			input:         "/",
			expectedError: true,
		},
		{
			input:         "some/random/path",
			expectedError: true,
		},
		{
			input:          "ClusterPolicy",
			expectedOutput: clusterPolicyResource,
		},
		{
			input:         "ClusterPolicy/",
			expectedError: true,
		},
		{
			input:         "folders/456/ClusterPolicy",
			expectedError: true,
		},
		{
			input:          "PolicyNode",
			expectedOutput: policyNodeResource,
		},
		{
			input:          "folders/456/PolicyNode",
			expectedOutput: policyNodeResource,
		},
		{
			input:         "folders//456/PolicyNode",
			expectedError: true,
		},
		{
			input:          "namespaces/456/PolicyNode",
			expectedOutput: policyNodeResource,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.input, func(t *testing.T) {
			output, err := policyResourceType(tc.input)
			if err != nil {
				if !tc.expectedError {
					t.Fatalf("Unexpected error: %v", err)
					return
				}
			} else {
				if tc.expectedError {
					t.Fatal("Expected error")
					return
				}
			}

			if diff := deep.Equal(output, tc.expectedOutput); diff != nil {
				t.Fatalf("Actual and expected output don't match: %v", diff)
			}
		})
	}

}
