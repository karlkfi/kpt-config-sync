package main

import (
	"bytes"
	"io/ioutil"
	authz "k8s.io/api/authorization/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestRequest(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(ServeFunc()))
	defer ts.Close()
	client := ts.Client()

	// The test setup approach was taken from:
	// https://github.com/kubernetes/kubernetes/blob/73326ef01d2d7c7eb3b8738ab57bf3b3d268ef97/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook_test.go#L504
	expTypeMeta := metav1.TypeMeta{
		APIVersion: "authorization.k8s.io/v1beta1",
		Kind:       "SubjectAccessReview",
	}

	tt := []struct {
		url         string
		contentType string
		request     authz.SubjectAccessReview
		expected    authz.SubjectAccessReview
		status      int
	}{
		{
			url:         ts.URL + "/authorize",
			contentType: "text/html",
			request:     authz.SubjectAccessReview{},
			expected:    authz.SubjectAccessReview{},
			status:      http.StatusUnsupportedMediaType,
		},
		{
			url:         ts.URL + "/authorize",
			contentType: "application/json",
			request: authz.SubjectAccessReview{
				TypeMeta: expTypeMeta,
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
				TypeMeta: expTypeMeta,
				Status: authz.SubjectAccessReviewStatus{
					Allowed: true,
				},
			},
			status: http.StatusOK,
		},
	}
	for _, ttt := range tt {
		actual, actualStatus := requestReview(t,
			client, ttt.url, ttt.contentType, ttt.request)
		if !reflect.DeepEqual(ttt.expected, actual) ||
			actualStatus != ttt.status {
			t.Errorf("Response: actual: '%+v', actualStatus: %+v,\nfor: '%+v'",
				actual, actualStatus, ttt)
		}
	}
}

// requestReview sends 'request' to 'url' with the given 'contentType'.  Returns
// the resulting request, and the return code.
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
