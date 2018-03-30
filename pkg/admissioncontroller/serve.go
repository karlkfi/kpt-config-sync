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

// functions for initializing and serving from the admission controller.
package admissioncontroller

import (
	"fmt"
	"time"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"net"

	"github.com/golang/glog"
	policynodeversions "github.com/google/nomos/pkg/client/informers/externalversions"
	informerspolicynodev1 "github.com/google/nomos/pkg/client/informers/externalversions/policyhierarchy/v1"
	policynodemeta "github.com/google/nomos/pkg/client/meta"
	"github.com/google/nomos/pkg/service"
	"github.com/pkg/errors"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	"k8s.io/client-go/rest"
)

// We have to wait for the endpoint to come up before self registering the webhook,
// otherwise the endpoint will never come up since the admission controller will appear down and
// block all requests, including the endpoint initialization
func WaitForEndpoint(clientset *kubernetes.Clientset, name, namespace string, timeout time.Duration) error {
	for t := time.Now(); time.Since(t) < timeout; time.Sleep(time.Second) {
		endpoint, err := clientset.CoreV1().Endpoints(namespace).Get(name, metav1.GetOptions{})

		if api_errors.IsNotFound(err) {
			glog.Info("Endpoint not ready yet...")
			continue
		}

		if err != nil {
			return errors.Wrap(err, "Failed while checking endpoint readiness")
		}

		if len(endpoint.Subsets) == 0 || len(endpoint.Subsets[0].Addresses) == 0 {
			glog.Info("Endpoint address not ready yet...")
			continue
		} else {
			glog.V(3).Info("Endpoint ready: ", endpoint)
			return nil
		}
	}
	return fmt.Errorf("timed out waiting for endpoint")
}

// GetAPIServerCert retrieves the client certificate used to sign requests from api-server.
//
// See --proxy-client-cert-file flag description:
// https://kubernetes.io/docs/admin/kube-apiserver
func GetAPIServerCert(clientset *kubernetes.Clientset) ([]byte, error) {
	c, err := clientset.CoreV1().ConfigMaps("kube-system").Get("extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get client auth from configmap: %v", err)
	}

	pem, ok := c.Data["requestheader-client-ca-file"]
	if !ok {
		return nil, fmt.Errorf("cannot find the ca.crt in the configmap, configMap.Data is %#v", c.Data)
	}
	return []byte(pem), nil
}

// GetWebhookClientAndCert returns a webhook client, which has cleared any existing configuration for this controller
// and the contents of the certificate file.
func GetWebhookClientAndCert(
	clientset *kubernetes.Clientset,
	caCertFile,
	webHookConfigName string) (v1beta1.ValidatingWebhookConfigurationInterface, []byte, error) {
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read ca bundle file")
	}
	client := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()
	_, err = client.Get(webHookConfigName, metav1.GetOptions{})
	if err == nil {
		glog.Infof("Deleting the existing ValidatingWebhookConfiguration")
		if err2 := client.Delete(webHookConfigName, nil); err2 != nil {
			return nil, nil, errors.Wrap(err2, "failed to delete ValidatingWebhookConfiguration")
		}
	} else if !api_errors.IsNotFound(err) {
		return nil, nil, errors.Wrap(err, "failed retrieving existing ValidatingWebhookConfiguration")
	}
	return client, caCert, nil
}

// Serve returns a handler function for the Admission controller.
func Serve(controller Admitter) service.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body == nil {
			glog.Error("empty request")
			http.Error(w, "empty request", http.StatusBadRequest)
			return
		}

		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}

		// verify the content type is accurate
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			glog.Errorf("contentType=%s, expect application/json", contentType)
			http.Error(w, fmt.Sprintf("Invalid content type %s", contentType), http.StatusBadRequest)
			return
		}

		review := admissionv1beta1.AdmissionReview{}
		if err := json.Unmarshal(body, &review); err != nil {
			glog.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		reviewStatus := controller.Admit(review)
		glog.Infof("Admission decision for namespace %s, object %s.%s %s: %v",
			review.Request.Namespace, review.Request.Kind.Kind, review.Request.Kind.Version, review.Request.Name, reviewStatus)
		ar := admissionv1beta1.AdmissionReview{
			Response: reviewStatus,
		}

		resp, err := json.Marshal(ar)
		if err != nil {
			glog.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(resp); err != nil {
			glog.Error(err)
		}
	}
}

func ServeTLS(server *http.Server, listener net.Listener, stopChannel chan struct{}) {
	err := server.ServeTLS(listener.(*net.TCPListener), "", "")

	if err != nil {
		glog.Fatal("Failed during serveTLS: ", err)
	}
	close(stopChannel)
}

// ServeFunc returns the serving function for this server.  Use for testing.
func ServeFunc(controller Admitter) service.HandlerFunc {
	return service.WithStrictTransport(service.NoCache(Serve(controller)))
}

// SetupPolicyNodeInformer returns a newly configured PolicyNodeInformer.
func SetupPolicyNodeInformer(config *rest.Config) (informerspolicynodev1.PolicyNodeInformer, error) {
	policyNodeClient, err := policynodemeta.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	policyNodeFactory := policynodeversions.NewSharedInformerFactory(
		policyNodeClient.PolicyHierarchy(), time.Minute,
	)
	policyNodeInformer := policyNodeFactory.Nomos().V1().PolicyNodes()
	policyNodeInformer.Informer()
	policyNodeFactory.Start(nil)

	return policyNodeInformer, nil
}
