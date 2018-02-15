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

// This package runs the hierarchical resource quota admission controller
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/admissioncontroller"
	policynodeversions "github.com/google/stolos/pkg/client/informers/externalversions"
	informerspolicynodev1 "github.com/google/stolos/pkg/client/informers/externalversions/policyhierarchy/v1"
	policynodemeta "github.com/google/stolos/pkg/client/meta"
	"github.com/google/stolos/pkg/service"
	"github.com/google/stolos/pkg/syncer/labeling"
	"github.com/google/stolos/pkg/util/log"
	"github.com/pkg/errors"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const externalAdmissionHookConfigName = "stolos-resource-quota"

// 5 seconds should be enough for endpoint to come up in the Kubernetes server.
const endpointRegistrationTimeout = time.Second * 5

var (
	caBundleFile = flag.String("ca-cert", "ca.crt", "Webhook server bundle cert used by api-server to authenticate the webhook server.")
	enablemTLS   = flag.Bool("enable-mutual-tls", false, "If set, enables mTLS verification of the client connecting to the admission controller.")
)

func serve(controller admissioncontroller.Admitter) service.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			if data, err := ioutil.ReadAll(r.Body); err == nil {
				body = data
			}
		}

		// verify the content type is accurate
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			glog.Errorf("contentType=%s, expect application/json", contentType)
			return
		}

		review := admissionv1beta1.AdmissionReview{}
		if err := json.Unmarshal(body, &review); err != nil {
			glog.Error(err)
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
		}
		if _, err := w.Write(resp); err != nil {
			glog.Error(err)
		}
	}
}

// ServeFunc returns the serving function for this server.  Use for testing.
func ServeFunc(controller admissioncontroller.Admitter) service.HandlerFunc {
	return service.WithStrictTransport(service.NoCache(serve(controller)))
}

func setupPolicyNodeInformer(config *rest.Config) (informerspolicynodev1.PolicyNodeInformer, error) {
	policyNodeClient, err := policynodemeta.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	policyNodeFactory := policynodeversions.NewSharedInformerFactory(
		policyNodeClient.PolicyHierarchy(), time.Minute,
	)
	policyNodeInformer := policyNodeFactory.Stolos().V1().PolicyNodes()
	policyNodeInformer.Informer()
	policyNodeFactory.Start(nil)

	return policyNodeInformer, nil
}

func setupResourceQuotaInformer(config *rest.Config) (informerscorev1.ResourceQuotaInformer, error) {
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	k8sFactory := informers.NewSharedInformerFactory(k8sClient, time.Minute)
	resourceQuotaInformer := k8sFactory.Core().V1().ResourceQuotas()
	resourceQuotaInformer.Informer()
	k8sFactory.Start(nil)

	return resourceQuotaInformer, nil
}

// getAPIServerCert retrieves the client certificate used to sign requests from api-server.
//
// See --proxy-client-cert-file flag description:
// https://kubernetes.io/docs/admin/kube-apiserver
func getAPIServerCert(clientset *kubernetes.Clientset) ([]byte, error) {
	c, err := clientset.CoreV1().ConfigMaps("kube-system").Get("extension-apiserver-authentication", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to get client auth from configmap: %v", err)
	}

	pem, ok := c.Data["requestheader-client-ca-file"]
	if !ok {
		return nil, fmt.Errorf("cannot find the ca.crt in the configmap, configMap.Data is %#v", c.Data)
	}
	return []byte(pem), nil
}

// register the webhook admission controller with the kube-apiserver.
func selfRegister(clientset *kubernetes.Clientset, caCertFile string) error {
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return fmt.Errorf("Failed to read ca bundle file: %v", err)
	}
	client := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()
	_, err = client.Get(externalAdmissionHookConfigName, metav1.GetOptions{})
	if err == nil {
		glog.Infof("Deleting the existing ValidatingWebhookConfiguration")
		if err2 := client.Delete(externalAdmissionHookConfigName, nil); err2 != nil {
			return fmt.Errorf("Failed to delete ValidatingWebhookConfiguration: %v", err2)
		}
	}
	failurePolicy := admissionregistrationv1beta1.Fail
	webhookConfig := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: externalAdmissionHookConfigName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name: "resourcequota.stolos.dev",
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update,
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{"*"},
						APIVersions: []string{"*"},
						Resources:   []string{"*"},
					},
				}},
				FailurePolicy: &failurePolicy,
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: "stolos-system",
						Name:      "resourcequota-admission-controller",
					},
					CABundle: caCert,
				},
				NamespaceSelector: metav1.SetAsLabelSelector(labeling.OriginLabel),
			},
		},
	}
	glog.Infof("Creating ValidatingWebhookConfiguration")
	if _, err := client.Create(webhookConfig); err != nil {
		return fmt.Errorf("Failed to create ValidatingWebhookConfiguration: %v", err)
	}
	return nil
}

// We have to wait for the endpoint to come up before self registering the webhook,
// otherwise the endpoint will never come up since the admission controller will appear down and
// block all requests, including the endpoint initialization
func waitForEndpoint(clientset *kubernetes.Clientset) error {
	for t := time.Now(); time.Since(t) < endpointRegistrationTimeout; time.Sleep(time.Second) {
		endpoint, err := clientset.CoreV1().Endpoints("stolos-system").Get(
			"resourcequota-admission-controller", metav1.GetOptions{})

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

func serveTLS(server *http.Server, listener net.Listener, stopChannel chan struct{}) {
	err := server.ServeTLS(listener.(*net.TCPListener), "", "")

	if err != nil {
		glog.Fatal("Failed during serveTLS: ", err)
	}
	close(stopChannel)
}

func main() {
	flag.Parse()
	log.Setup()

	glog.Info("Hierarchical Resource Quota Admission Controller starting up")

	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatal("Failed to load in cluster config: ", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal("Failed to create client set: ", err)
	}
	var clientCert []byte
	if *enablemTLS {
		clientCert, err = getAPIServerCert(clientset)
		if err != nil {
			glog.Fatal("Failed to get client cert: ", err)
		}
	}
	policyNodeInformer, err := setupPolicyNodeInformer(config)
	if err != nil {
		glog.Fatal("Failed setting up policyNode informer: ", err)
	}
	resourceQuotaInformer, err := setupResourceQuotaInformer(config)
	if err != nil {
		glog.Fatal("Failed setting up resourceQuota informer: ", err)
	}
	glog.Info("Waiting for informers to sync...")
	if !cache.WaitForCacheSync(nil, policyNodeInformer.Informer().HasSynced, resourceQuotaInformer.Informer().HasSynced) {
		glog.Fatal("Failure while waiting for informers to sync")
	}

	go service.ServeMetrics()

	server := service.Server(
		ServeFunc(
			admissioncontroller.NewResourceQuotaAdmitter(
				policyNodeInformer, resourceQuotaInformer)), clientCert)

	stopChannel := make(chan struct{})
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		glog.Fatal("Failed to start https listener: ", err)
	}
	defer listener.Close()

	glog.Info("Server listening at: ", server.Addr)

	go serveTLS(server, listener, stopChannel)

	// Wait for endpoint to come up before self-registering
	err = waitForEndpoint(clientset)
	if err != nil {
		glog.Fatal("Failed waiting for endpoint: ", err)
	}

	// Finally register the webhook to block admission according to quota policy
	if err := selfRegister(clientset, *caBundleFile); err != nil {
		glog.Fatal("Failed to register webhook: ", err)
	}

	<-stopChannel
}
