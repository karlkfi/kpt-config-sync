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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	policyhierarchy "github.com/google/stolos/pkg/api/policyhierarchy/v1"
	"github.com/google/stolos/pkg/authorizer"
	authz "k8s.io/api/authorization/v1beta1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
)

type testCase struct {
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
}

func newTestServer(policynodes ...runtime.Object) *httptest.Server {
	return httptest.NewTLSServer(http.HandlerFunc(ServeFunc(
		authorizer.New(authorizer.NewTestInformer(policynodes...)))))
}

// testRequest is a repeating test case. 't' is the test harness, 'policynodes'
// are the policy node objects injected in the fake policy nodes hierarchy, and
// tt is the test case to run.
func testRequest(name string, t *testing.T, policynodes *[]runtime.Object, tt *[]testCase) {
	// The objects listed here will appear as if they have
	// already been present in the storage that the test
	// client is reading from.
	ts := newTestServer(*policynodes...)
	defer ts.Close()
	for i, ttt := range *tt {
		actual, actualStatus := requestReview(t,
			ts.Client(), ts.URL, ttt.contentType, ttt.request)
		if !reflect.DeepEqual(ttt.expected, actual) ||
			actualStatus != ttt.expectedStatusCode {
			t.Errorf("[%v:%v] Actual:\n%v\nactualStatus:\n%v\nfor:\n%v",
				name, i, spew.Sdump(actual), spew.Sdump(actualStatus),
				spew.Sdump(ttt))
		}
	}
}

// TestSimpleRequest tests namespace finding in a simple policy nodes
// hierarchy.
func TestSimpleRequest(t *testing.T) {
	policynodes := []runtime.Object{
		&policyhierarchy.PolicyNode{
			ObjectMeta: meta.ObjectMeta{
				Name: "kittiesandponies",
			},
		},
	}
	// The test setup approach was taken from:
	// https://github.com/kubernetes/kubernetes/blob/73326ef01d2d7c7eb3b8738ab57bf3b3d268ef97/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook_test.go#L504
	tt := []testCase{
		{
			contentType:        "text/html",
			request:            authz.SubjectAccessReview{},
			expected:           authz.SubjectAccessReview{},
			expectedStatusCode: http.StatusUnsupportedMediaType,
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
					EvaluationError: "while getting hierarchical policy rules: while getting policy node: onlyponies: partial policy rules, missing namespace: onlyponies",
				},
			},
			expectedStatusCode: http.StatusOK,
		},
	}
	testRequest("TestSimpleRequest", t, &policynodes, &tt)
}

func TestTreeHierarchy(t *testing.T) {
	// animals
	// `- kittiesandponies
	//    |- kitties
	//    `- ponies
	policynodes := []runtime.Object{
		&policyhierarchy.PolicyNode{
			ObjectMeta: meta.ObjectMeta{
				Name: "animals",
			},
			Spec: policyhierarchy.PolicyNodeSpec{
				Parent: "",
			},
		},
		&policyhierarchy.PolicyNode{
			ObjectMeta: meta.ObjectMeta{
				Name: "kittiesandponies",
			},
			Spec: policyhierarchy.PolicyNodeSpec{
				Parent: "animals",
			},
		},
		&policyhierarchy.PolicyNode{
			ObjectMeta: meta.ObjectMeta{
				Name: "kitties",
			},
			Spec: policyhierarchy.PolicyNodeSpec{
				Parent: "kittiesandponies",
			},
		},
		&policyhierarchy.PolicyNode{
			ObjectMeta: meta.ObjectMeta{
				Name: "ponies",
			},
			Spec: policyhierarchy.PolicyNodeSpec{
				Parent: "kittiesandponies",
			},
		},
	}
	// The test setup approach was taken from:
	// https://github.com/kubernetes/kubernetes/blob/73326ef01d2d7c7eb3b8738ab57bf3b3d268ef97/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook_test.go#L504
	tt := []testCase{
		{
			contentType: "application/json",
			request: authz.SubjectAccessReview{
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
						Namespace:   "kitties",
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
						Namespace:   "ponies",
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
					EvaluationError: "while getting hierarchical policy rules: while getting policy node: onlyponies: partial policy rules, missing namespace: onlyponies",
				},
			},
			expectedStatusCode: http.StatusOK,
		},
	}
	testRequest("TestTreeHierarchy", t, &policynodes, &tt)

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
