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

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/gogo/status"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	watcher "github.com/google/nomos/clientgen/watcher/v1"
	mock "github.com/google/nomos/clientgen/watcher/v1/testing"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	clientaction "github.com/google/nomos/pkg/client/action"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/actions"
	"github.com/google/nomos/pkg/util/policynode"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
			},
		},
		{
			testName: "Initial state org with ClusterPolicy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "ClusterPolicy", State: watcher.Change_EXISTS, Continued: false, Data: orgCPProto},
			},
			expectedActions: []string{
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
			},
		},
		{
			testName: "Initial state with non-existent root",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
			},
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
			},
		},
		{
			testName: "Initial state with non-existent resource",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: true, Data: emptyProto},
				{Element: "PolicyNode", State: watcher.Change_DOES_NOT_EXIST, Continued: false, Data: emptyProto},
			},
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1/PolicyNodes/backend/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1/PolicyNodes/backend/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1/PolicyNodes/backend/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
				"nomos.dev/v1/ClusterPolicies/nomos-cluster-policy/delete",
			},
		},
		{
			testName: "Incremental change includes root",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
			},
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1/PolicyNodes/backend/create",
			},
		},
		{
			testName: "Incremental state not valid hierarchy",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto},
				{Element: "folders/456/PolicyNode", State: watcher.Change_EXISTS, Continued: false, Data: folderPNProto},
			},
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
				"nomos.dev/v1/PolicyNodes/organization-123/update",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
				"nomos.dev/v1/PolicyNodes/organization-123/update",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
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
				"nomos.dev/v1/PolicyNodes/organization-123/create",
				"nomos.dev/v1alpha1/Syncs/rbac/create",
				"nomos.dev/v1/PolicyNodes/folder-456/create",
				"nomos.dev/v1/PolicyNodes/backend/create",
			},
		},
		{
			testName: "Resume marker passed",
			batch1: []*watcher.Change{
				{Element: "", State: watcher.Change_EXISTS, Continued: false, Data: emptyProto, ResumeMarker: []byte("token")},
			},
			expectedResumeMarker: []byte("token"),
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
			},
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
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
			},
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
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
			},
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
				"nomos.dev/v1alpha1/Syncs/rbac/create",
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
			expectedActions: []string{
				"nomos.dev/v1alpha1/Syncs/rbac/create",
			},
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
			recordAction := func(a clientaction.Interface) error {
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
				actions.NewFactories(nil, nil, nil, nil, nil),
				nameMap, len(tc.resumeMarker) != 0,
				cancel,
				time.Minute}

			resumeMarker, err := p.process()

			if (err != nil) != tc.expectedError {
				t.Fatalf("Err state is %v, wanted %v. Actual error: %s", !tc.expectedError, tc.expectedError, err)
			}

			if diff := cmp.Diff(resumeMarker, tc.expectedResumeMarker); diff != "" {
				t.Errorf("Resume marker is %v, wanted %v", resumeMarker, tc.expectedResumeMarker)
			}

			if diff := cmp.Diff(actualActions, tc.expectedActions); diff != "" {
				t.Errorf("Actual and expected actions don't match: %v\n%v\n%v", diff, actualActions, tc.expectedActions)
			}
		})
	}
}

type atomicGroupTestCase struct {
	name     string
	changes  map[string]*watcher.Change
	expected *v1.AllPolicies
}

func marshalOrFail(t *testing.T, pb proto.Message) *ptypes.Any {
	any, err := ptypes.MarshalAny(pb)
	if err != nil {
		t.Fatalf("failed to marshal proto %#v: %v", pb, err)
	}
	return any
}

func toResources(gvk schema.GroupVersionKind, os ...runtime.Object) []v1.GenericResources {
	grs := []v1.GenericResources{
		{
			Group: gvk.Group,
			Kind:  gvk.Kind,
			Versions: []v1.GenericVersionResources{
				{
					Version: gvk.Version,
					Objects: []runtime.RawExtension{},
				},
			},
		},
	}
	for _, o := range os {
		grs[0].Versions[0].Objects = append(grs[0].Versions[0].Objects, runtime.RawExtension{Object: o})
	}
	return grs
}

// TestAtomicGroup tests the private processAtomicGroup method to more easily test the AllPolicies
// object generated from the Watcher changes. Without this, it would be necessary to dig through the
// reflective actions.
func TestProcessAtomicGroup(t *testing.T) {
	testCases := []atomicGroupTestCase{
		{
			name: "RoleBinding",
			changes: map[string]*watcher.Change{
				"": {},
				"PolicyNode": {
					Element: "PolicyNode",
					State:   watcher.Change_EXISTS,
					Data: marshalOrFail(t, policynode.NewPolicyNode("foo", &v1.PolicyNodeSpec{
						Type: v1.Policyspace,
						RoleBindingsV1: []rbacv1.RoleBinding{{
							ObjectMeta: metav1.ObjectMeta{
								Name: "myrb",
							},
						}},
					})),
				},
			},
			expected: &v1.AllPolicies{
				PolicyNodes: map[string]v1.PolicyNode{
					"foo": {
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec: v1.PolicyNodeSpec{
							Type: v1.Policyspace,
							RoleBindingsV1: []rbacv1.RoleBinding{
								{ObjectMeta: metav1.ObjectMeta{Name: "myrb"}},
							},
							Resources: toResources(kinds.RoleBinding(),
								runtime.Object(&rbacv1.RoleBinding{
									ObjectMeta: metav1.ObjectMeta{
										Name: "myrb",
									}})),
						}},
				},
			},
		},
		{
			name: "Two RoleBindings",
			changes: map[string]*watcher.Change{
				"": {},
				"PolicyNode": {
					Element: "PolicyNode",
					State:   watcher.Change_EXISTS,
					Data: marshalOrFail(t, policynode.NewPolicyNode("foo", &v1.PolicyNodeSpec{
						Type: v1.Policyspace,
						RoleBindingsV1: []rbacv1.RoleBinding{
							{ObjectMeta: metav1.ObjectMeta{Name: "myrb1"}},
							{ObjectMeta: metav1.ObjectMeta{Name: "myrb2"}},
						},
					})),
				},
			},
			expected: &v1.AllPolicies{
				PolicyNodes: map[string]v1.PolicyNode{
					"foo": {
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec: v1.PolicyNodeSpec{
							Type: v1.Policyspace,
							RoleBindingsV1: []rbacv1.RoleBinding{
								{ObjectMeta: metav1.ObjectMeta{Name: "myrb1"}},
								{ObjectMeta: metav1.ObjectMeta{Name: "myrb2"}},
							},
							Resources: toResources(kinds.RoleBinding(),
								runtime.Object(&rbacv1.RoleBinding{
									ObjectMeta: metav1.ObjectMeta{Name: "myrb1"},
								}),
								runtime.Object(&rbacv1.RoleBinding{
									ObjectMeta: metav1.ObjectMeta{Name: "myrb2"},
								})),
						}},
				},
			},
		},
		{
			name: "ResourceQuota",
			changes: map[string]*watcher.Change{
				"": {},
				"PolicyNode": {
					Element: "PolicyNode",
					State:   watcher.Change_EXISTS,
					Data: marshalOrFail(t, policynode.NewPolicyNode("foo", &v1.PolicyNodeSpec{
						Type: v1.Policyspace,
						ResourceQuotaV1: &corev1.ResourceQuota{
							ObjectMeta: metav1.ObjectMeta{
								Name: "myrq",
							},
						},
					})),
				},
			},
			expected: &v1.AllPolicies{
				PolicyNodes: map[string]v1.PolicyNode{
					"foo": {
						ObjectMeta: metav1.ObjectMeta{
							Name: "foo",
						},
						Spec: v1.PolicyNodeSpec{
							Type: v1.Policyspace,
							ResourceQuotaV1: &corev1.ResourceQuota{
								ObjectMeta: metav1.ObjectMeta{
									Name: "myrq",
								},
							},
							Resources: toResources(kinds.ResourceQuota(),
								runtime.Object(&corev1.ResourceQuota{
									ObjectMeta: metav1.ObjectMeta{
										Name: "myrq",
									}})),
						}},
				},
			},
		},
		{
			name: "ClusterRole",
			changes: map[string]*watcher.Change{
				"": {},
				"ClusterPolicy": {
					Element: "ClusterPolicy",
					State:   watcher.Change_EXISTS,
					Data: marshalOrFail(t, policynode.NewClusterPolicy("nomos-cluster-policy",
						&v1.ClusterPolicySpec{
							ClusterRolesV1: []rbacv1.ClusterRole{{
								ObjectMeta: metav1.ObjectMeta{
									Name: "mycr",
								},
							}},
						})),
				},
			},
			expected: &v1.AllPolicies{
				ClusterPolicy: &v1.ClusterPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nomos-cluster-policy",
					},
					Spec: v1.ClusterPolicySpec{
						ClusterRolesV1: []rbacv1.ClusterRole{{
							ObjectMeta: metav1.ObjectMeta{
								Name: "mycr",
							},
						}},
						Resources: toResources(kinds.ClusterRole(),
							runtime.Object(&rbacv1.ClusterRole{
								ObjectMeta: metav1.ObjectMeta{
									Name: "mycr",
								},
							})),
					}},
				PolicyNodes: map[string]v1.PolicyNode{},
			},
		},
		{
			name: "ClusterRoleBinding",
			changes: map[string]*watcher.Change{
				"": {},
				"ClusterPolicy": {
					Element: "ClusterPolicy",
					State:   watcher.Change_EXISTS,
					Data: marshalOrFail(t, policynode.NewClusterPolicy("nomos-cluster-policy",
						&v1.ClusterPolicySpec{
							ClusterRoleBindingsV1: []rbacv1.ClusterRoleBinding{{
								ObjectMeta: metav1.ObjectMeta{
									Name: "mycrb",
								},
							}},
						})),
				},
			},
			expected: &v1.AllPolicies{
				ClusterPolicy: &v1.ClusterPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nomos-cluster-policy",
					},
					Spec: v1.ClusterPolicySpec{
						ClusterRoleBindingsV1: []rbacv1.ClusterRoleBinding{{
							ObjectMeta: metav1.ObjectMeta{
								Name: "mycrb",
							},
						}},
						Resources: toResources(kinds.ClusterRoleBinding(),
							runtime.Object(&rbacv1.ClusterRoleBinding{
								ObjectMeta: metav1.ObjectMeta{
									Name: "mycrb",
								},
							})),
					}},
				PolicyNodes: map[string]v1.PolicyNode{},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := watchProcessor{
				gcpToK8SName: ToK8SNameMap{},
			}
			actual, err := p.processAtomicGroup(tc.changes)
			if err != nil {
				t.Errorf("processAtomicGroup failed: %v", err)
			}
			if diff := cmp.Diff(actual, tc.expected); diff != "" {
				t.Errorf("actual and expected policies don't match:\n%s", cmp.Diff(actual, tc.expected))
			}
		})
	}
}

func TestRecvErr(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stream := mock.NewMockWatcher_WatchClient(ctrl)
	stream.EXPECT().Recv().Return(nil, errors.New("receive error"))

	a := func(_ clientaction.Interface) error {
		return nil
	}
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := watchProcessor{
		stream,
		a,
		v1.AllPolicies{},
		actions.NewFactories(nil, nil, nil, nil, nil),
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
	if diff := cmp.Diff(errors.Cause(err).Error(), expectedErr.Error()); diff != "" {
		t.Errorf("Actual and expected errors don't match: %v", diff)
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
	a := func(_ clientaction.Interface) error {
		return nil
	}
	defer cancel()
	p := watchProcessor{
		stream,
		a,
		v1.AllPolicies{},
		actions.NewFactories(nil, nil, nil, nil, nil),
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

func createResourceQuota(name string, namespace string) *corev1.ResourceQuota {
	return &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{"pods": resource.MustParse("10")},
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

			if diff := cmp.Diff(output, tc.expectedOutput); diff != "" {
				t.Errorf("Actual and expected output don't match: %v", diff)
			}
		})
	}

}
