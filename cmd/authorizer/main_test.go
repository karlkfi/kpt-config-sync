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

package main

import (
	"bytes"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/api/policyhierarchy/v1"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/authorizer"
	"github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy/fake"
	k8usv1 "github.com/mdruskin/kubernetes-enterprise-control/pkg/client/policyhierarchy/typed/k8us/v1"
	"io/ioutil"
	authz "k8s.io/api/authorization/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRequest(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(ServeFunc(
		authorizer.New(newPolicyHierarchyClient(
			// The objects listed here will appear as if they have
			// already been present in the storage that the test
			// client is reading from.
			&v1.PolicyNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kittiesandponies",
				},
			},
		)),
	)))
	defer ts.Close()

	// The test setup approach was taken from:
	// https://github.com/kubernetes/kubernetes/blob/73326ef01d2d7c7eb3b8738ab57bf3b3d268ef97/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook_test.go#L504
	tt := []struct {
		// The ContentType header of the request to be sent.
		contentType string
		// The webhook authorizer request.  request.ResourceAttributes
		// contains the request under test.
		request authz.SubjectAccessReview
		// The expected authz response.  expected.Status contains the
		// authorizer response.
		expected authz.SubjectAccessReview
		// The http status code reported.
		expectedStatusCode int
	}{
		{
			contentType:        "text/html",
			request:            authz.SubjectAccessReview{},
			expected:           authz.SubjectAccessReview{},
			expectedStatusCode: http.StatusUnsupportedMediaType,
		},
		{
			contentType: "application/json",
			request: authz.SubjectAccessReview{
				// From main.go.
				TypeMeta: authorizer.TypeMeta,
				Spec: authz.SubjectAccessReviewSpec{
					User:   "jane",
					Groups: []string{"group1", "group2"},
					ResourceAttributes: &authz.ResourceAttributes{
						Verb:        "GET",
						Namespace:   "kittiesandponies",
						Group:       "group3",
						Version:     "v7beta3",
						Resource:    "pods",
						Subresource: "proxy",
						Name:        "my-pod",
					},
				},
			},
			expected: authz.SubjectAccessReview{
				TypeMeta: authorizer.TypeMeta,
				Status: authz.SubjectAccessReviewStatus{
					Allowed: true,
				},
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			contentType: "application/json",
			request: authz.SubjectAccessReview{
				TypeMeta: authorizer.TypeMeta,
				Spec: authz.SubjectAccessReviewSpec{
					User:   "jane",
					Groups: []string{"group1", "group2"},
					ResourceAttributes: &authz.ResourceAttributes{
						Verb:        "GET",
						Namespace:   "onlyponies",
						Group:       "group3",
						Version:     "v7beta3",
						Resource:    "pods",
						Subresource: "proxy",
						Name:        "my-pod",
					},
				},
			},
			expected: authz.SubjectAccessReview{
				TypeMeta: authorizer.TypeMeta,
				Status: authz.SubjectAccessReviewStatus{
					Allowed:         false,
					EvaluationError: "while getting hierarchical policy rules: while getting policy node: onlyponies: policynodes.k8us.k8s.io \"onlyponies\" not found",
				},
			},
			expectedStatusCode: http.StatusOK,
		},
	}
	for _, ttt := range tt {
		actual, actualStatus := requestReview(t,
			ts.Client(), ts.URL, ttt.contentType, ttt.request)
		if !reflect.DeepEqual(ttt.expected, actual) ||
			actualStatus != ttt.expectedStatusCode {
			t.Errorf("Actual:\n%+v\nactualStatus:\n%+v\nfor:\n%+v",
				actual, actualStatus, ttt)
		}
	}
}

// newPolicyHierarchyClient creates a new fake policyhierarchy client, injecting
// 'presets' into the backing store.  Later calls on the client will pretend as
// if those objects were already inserted.
func newPolicyHierarchyClient(presets ...runtime.Object) k8usv1.K8usV1Interface {
	// fake.Clientset (which you get from fake.NewSimpleClientSet) is not
	// interface-compatible with the "real" client set.  But, the clients
	// are.  So we inject them instead.
	return fake.NewSimpleClientset(presets...).K8usV1()
}

// requestReview sends 'request' to 'url' with the given 'contentType'.  Returns
// the resulting response, and the returned HTTP status code.
func requestReview(
	t *testing.T,
	client *http.Client, url string, contentType string,
	request authz.SubjectAccessReview) (authz.SubjectAccessReview, int) {

	requestBytes, err := json.Marshal(request)
	if err != nil {
		t.Errorf("json.Marshal: %+v, +%v", request, err)
	}
	resp, err := client.Post(url, contentType, bytes.NewReader(requestBytes))
	if err != nil {
		t.Errorf("client.Post: %+v, +%v", request, err)
	}

	// Return early if the server set the status code.
	if resp.StatusCode != http.StatusOK {
		return authz.SubjectAccessReview{}, resp.StatusCode
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll: %+v: %+v", url, err)
	}

	var respStruct authz.SubjectAccessReview
	err = json.Unmarshal(respBytes, &respStruct)
	if err != nil {
		t.Errorf("json.Unmarshal: %+v: %+v", url, err)
	}
	return respStruct, resp.StatusCode
}
