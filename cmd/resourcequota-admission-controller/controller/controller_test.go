package controller

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/admissioncontroller/resourcequota"
	"github.com/google/nomos/pkg/testing/fakeinformers"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/util/json"
)

// nolint:deadcode
func TestRequest(t *testing.T) {
	ts := httptest.NewTLSServer(admissioncontroller.ServeFunc(resourcequota.NewAdmitter(
		fakeinformers.NewResourceQuotaInformer(), fakeinformers.NewHierarchicalQuotaInformer())))
	defer ts.Close()

	request := admissionv1beta1.AdmissionReview{
		Request: &admissionv1beta1.AdmissionRequest{},
	}
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

	var respStruct admissionv1beta1.AdmissionReview
	err = json.Unmarshal(respBytes, &respStruct)
	if err != nil {
		t.Errorf("json.Unmarshal: %+v: %+v", ts.URL, err)
	}

	if !respStruct.Response.Allowed {
		t.Errorf("Unexpected response status %v", respStruct)
	}
}
