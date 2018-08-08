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
	"io"
	"testing"
	"time"

	"github.com/go-test/deep"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/gogo/status"
	"github.com/golang/mock/gomock"
	watcher "github.com/google/nomos/clientgen/watcher/v1"
	mock "github.com/google/nomos/clientgen/watcher/v1/testing"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	client_action "github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/policyimporter/actions"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Example PolicyNode resources
	orgPN, orgPNUpdated, orgPNUpdatedLabels, nsPN, folderPN *v1.PolicyNode
	// Example any proto marshalled policies resources.
	emptyProto, orgPNProto, orgPNProtoUpdated, orgPNProtoUpdatedLabels, orgCPProto, folderPNProto, nsPNProto *ptypes.Any
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
	// The expected resume marker.
	expectedResumeMarker []byte
	// The resume marker to send.
	resumeMarker []byte
	nameMap      ToK8SNameMap
}

func init() {
	emptyProto = &ptypes.Any{}
	anError, err := ptypes.MarshalAny(status.New(codes.DeadlineExceeded, "deadline exceeded when receiving policies").Proto())
	if err != nil {
		panic(err)
	}
	orgPN = newPolicyNode("organization-123", "", true)
	orgPNProto = toAnyProto(orgPN)
	orgPNUpdated = orgPN.DeepCopy()
	orgPNUpdated.Spec.ResourceQuotaV1 = createResourceQuota("my-quota", "organization-123")
	orgPNProtoUpdated = toAnyProto(orgPNUpdated)
	orgPNUpdatedLabels = orgPN.DeepCopy()
	orgPNUpdatedLabels.Labels = map[string]string{"env": "prod"}
	orgPNProtoUpdatedLabels = toAnyProto(orgPNUpdatedLabels)
	orgCPProto = toAnyProto(newClusterPolicy("organization-123"))
	folderPN = newPolicyNode("folder-456", "organization-123", true)
	folderPNProto = toAnyProto(folderPN)
	nsPN = newPolicyNode("backend", "folder-456", false)
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
			testName: "Initial state with non-existent root",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
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
				{Element: "projects/789/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: nsPNProto},
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
				{Element: "projects/789/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: nsPNProto},
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
		{
			testName: "Initial state with unmarshal error",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "ClusterPolicy", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			expectedError: true,
		},
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
			testName: "Incremental change with valid hierarchy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: folderPNProto},
				{Element: "projects/789/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: nsPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/backend/upsert",
			},
		},
		{
			testName: "Incremental change delete PolicyNode",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: folderPNProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: true, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/delete",
				"nomos.dev/v1/PolicyNodes/organization-123/delete",
			},
		},
		{
			testName: "Incremental change delete non-existent PolicyNode",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: folderPNProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "folders/999/PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/delete",
			},
		},
		{
			testName: "Incremental change delete existing PolicyNode",
			batch1: []*watcher.Change{
				{Element: "folders/456/PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
			},
			nameMap:      ToK8SNameMap{"folders/456/PolicyNode": "folder-456"},
			resumeMarker: []byte("hello"),
			currentPolicies: v1.AllPolicies{PolicyNodes: map[string]v1.PolicyNode{
				"organization-123": *orgPN,
				"folder-456":       *folderPN,
			}},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/folder-456/delete",
			},
		},
		{
			testName: "Incremental change delete ClusterPolicy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "ClusterPolicy", State: watcher.Change_EXISTS, Continued: false, Data: orgCPProto},
				{Element: "ClusterPolicy", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/upsert",
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/delete",
			},
		},
		{
			testName: "Incremental change includes root",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
			},
		},
		{
			testName: "Incremental atomic group not in tree invariant order",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "projects/789/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: nsPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/backend/upsert",
			},
		},
		{
			testName: "Incremental state not valid hierarchy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
			},
			expectedError: true,
		},
		{
			testName: "Incremental change update policy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProtoUpdated},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
			},
		},
		{
			testName: "Incremental change update label",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProtoUpdatedLabels},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
			},
		},
		{
			testName: "Incremental change no update",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
			},
		},
		{
			testName: "Incremental atomic group duplicates",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: true, Data: folderPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
			},
		},
		{
			testName: "Incremental change multiple atomic groups",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
				{Element: "projects/789/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: nsPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/organization-123/upsert",
				"nomos.dev/v1/PolicyNodes/folder-456/upsert",
				"nomos.dev/v1/PolicyNodes/backend/upsert",
			},
		},
		{
			testName: "Resume marker passed",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto, ResumeMarker: []byte("token")},
			},
			expectedResumeMarker: []byte("token"),
		},
		{
			testName: "Resume marker not clobbered by null",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto, ResumeMarker: []byte("token")},
			},
			batch2: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
			},
			expectedResumeMarker: []byte("token"),
		},
		{
			testName: "Resume marker clobbered by new marker",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto, ResumeMarker: []byte("token")},
			},
			batch2: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto, ResumeMarker: []byte("token2")},
			},
			expectedResumeMarker: []byte("token2"),
		},
		{
			testName: "Initial policies wiped without resume token",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
			},
			currentPolicies: v1.AllPolicies{PolicyNodes: map[string]v1.PolicyNode{
				"organization-123": *orgPN,
				"folder-456":       *folderPN,
			}},
			expectedActions: []string{
				"nomos.dev/v1/PolicyNodes/folder-456/delete",
				"nomos.dev/v1/PolicyNodes/organization-123/delete",
			},
		},
		{
			testName: "Initial policies not wiped with resume token",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: orgPNProto},
			},
			currentPolicies: v1.AllPolicies{PolicyNodes: map[string]v1.PolicyNode{
				"organization-123": *orgPN,
				"folder-456":       *folderPN,
			}},
			resumeMarker: []byte("hello"),
			// No actions expected
		},
	}
}

func TestGen(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			stream := mock.NewMockWatcher_WatchClient(ctrl)

			stream.EXPECT().Recv().Return(&watcher.ChangeBatch{Changes: tc.batch1}, nil)
			if tc.batch2 != nil {
				stream.EXPECT().Recv().Return(&watcher.ChangeBatch{Changes: tc.batch2}, nil)
			}
			if !tc.expectedError {
				stream.EXPECT().Recv().Return(nil, io.EOF)
			}

			var actualActions []string
			recordAction := func(a client_action.Interface) error {
				actualActions = append(actualActions, a.String())
				return nil
			}
			_, cancel := context.WithCancel(context.Background())
			defer cancel()
			// Initialize the map to nonempty only for test cases that specify a nonempty one.
			nameMap := tc.nameMap
			if nameMap == nil {
				nameMap = ToK8SNameMap(map[string]string{})
			}
			// Factories take nil arguments since we don't need to apply the actions for these tests.
			p := watchProcessor{
				stream,
				recordAction,
				tc.currentPolicies,
				actions.NewFactories(nil, nil, nil),
				nameMap, len(tc.resumeMarker) != 0,
				cancel,
				time.Minute}

			resumeMarker, err := p.process()

			if (err != nil) != tc.expectedError {
				t.Fatalf("Err state is %v, wanted %v. Actual error: %s", !tc.expectedError, tc.expectedError, err)
			}

			if diff := deep.Equal(resumeMarker, tc.expectedResumeMarker); diff != nil {
				t.Fatalf("Resume marker is %v, wanted %v", resumeMarker, tc.expectedResumeMarker)
			}

			if diff := deep.Equal(actualActions, tc.expectedActions); diff != nil {
				t.Fatalf("Actual and expected actions don't match: %v\n%v\n%v", diff, actualActions, tc.expectedActions)
			}
		})
	}
}

func TestRecvErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stream := mock.NewMockWatcher_WatchClient(ctrl)
	stream.EXPECT().Recv().Return(nil, errors.New("receive error"))

	a := func(_ client_action.Interface) error {
		return nil
	}
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := watchProcessor{
		stream,
		a,
		v1.AllPolicies{},
		actions.NewFactories(nil, nil, nil),
		ToK8SNameMap(map[string]string{}),
		false,
		cancel,
		time.Minute,
	}

	_, err := p.process()
	expectedErr := errors.New("receive error")
	if err == nil {
		t.Fatalf("Expected error")
	}
	if diff := deep.Equal(errors.Cause(err), expectedErr); diff != nil {
		t.Fatalf("Actual and expected errors don't match: %v", diff)
	}
}

// Test that generate() returns when context is done.
func TestTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	stream := mock.NewMockWatcher_WatchClient(ctrl)
	stream.EXPECT().Recv().DoAndReturn(func() (*watcher.ChangeBatch, error) {
		<-ctx.Done()
		return &watcher.ChangeBatch{}, status.Error(codes.Canceled, "canceled")
	}).MinTimes(0)
	a := func(_ client_action.Interface) error {
		return nil
	}
	defer cancel()
	p := watchProcessor{
		stream,
		a,
		v1.AllPolicies{},
		actions.NewFactories(nil, nil, nil),
		ToK8SNameMap{},
		false,
		cancel,
		time.Nanosecond,
	}

	_, _ = p.process()
}

func newPolicyNode(name string, parent string, policyspace bool) *v1.PolicyNode {
	pnt := v1.Namespace
	if policyspace {
		pnt = v1.Policyspace
	}
	pn := policynode.NewPolicyNode(name,
		&v1.PolicyNodeSpec{
			Type:   pnt,
			Parent: parent,
		})
	return pn
}

func newClusterPolicy(name string) *v1.ClusterPolicy {
	return policynode.NewClusterPolicy(name, &v1.ClusterPolicySpec{})
}

func toAnyProto(m interface{}) *ptypes.Any {
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

func createResourceQuota(name string, namespace string) *core_v1.ResourceQuota {
	return &core_v1.ResourceQuota{
		TypeMeta: meta_v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: core_v1.ResourceQuotaSpec{
			Hard: core_v1.ResourceList{"pods": resource.MustParse("10")},
		},
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
			input:          "projects/456/PolicyNode",
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
