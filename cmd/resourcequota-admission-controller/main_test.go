/*
Copyright 2017 The Stolos Authors.
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
	"github.com/google/stolos/pkg/admission-controller"
	"github.com/google/stolos/pkg/testing/fakeinformers"
	"io/ioutil"
	admissionv1alpha1 "k8s.io/api/admission/v1alpha1"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequest(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(ServeFunc(admission_controller.NewResourceQuotaAdmitter(
		fakeinformers.NewPolicyNodeInformer(), fakeinformers.NewResourceQuotaInformer()))))
	defer ts.Close()

	request := admissionv1alpha1.AdmissionReview{}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		t.Errorf("json.Marshal: %+v, +%v", request, err)
	}
	resp, err := ts.Client().Post(ts.URL, "application/json", bytes.NewReader(requestBytes))
	if err != nil {
		t.Errorf("client.Post: %+v, +%v", request, err)
	}

	// Return early if the server set the status code.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code %v", resp.StatusCode)
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		t.Errorf("ioutil.ReadAll: %+v: %+v", ts.URL, err)
	}

	var respStruct admissionv1alpha1.AdmissionReview
	err = json.Unmarshal(respBytes, &respStruct)
	if err != nil {
		t.Errorf("json.Unmarshal: %+v: %+v", ts.URL, err)
	}

	if !respStruct.Status.Allowed {
		t.Errorf("Unexpected response status %v", respStruct)
	}
}
