/*
Copyright 2018 The Nomos Authors.
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

// This package runs the policy validation admission controller
package main

import (
	"flag"
	"net"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/admissioncontroller/policy"
	"github.com/google/nomos/pkg/generic-syncer/labeling"

	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"github.com/pkg/errors"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	externalAdmissionHookConfigName = "policy.nomos.dev"
	controllerNamespace             = "nomos-system"
	controllerName                  = "policy-admission-controller"
)

var (
	caBundleFile = flag.String("ca-cert", "ca.crt", "Webhook server bundle cert used by api-server to authenticate the webhook server.")
	enablemTLS   = flag.Bool("enable-mutual-tls", false, "If set, enables mTLS verification of the client connecting to the admission controller.")
)

// register the webhook admission controller with the kube-apiserver.
func selfRegister(clientset *kubernetes.Clientset, caCertFile string) error {
	caCert, err := admissioncontroller.GetWebhookCert(caCertFile)
	if err != nil {
		return err
	}

	failurePolicy := admissionregistrationv1beta1.Fail
	webhookConfig := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   externalAdmissionHookConfigName,
			Labels: labeling.NomosSystem.New(),
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name: externalAdmissionHookConfigName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1beta1.OperationType{
							admissionregistrationv1beta1.Create,
							admissionregistrationv1beta1.Update,
							admissionregistrationv1beta1.Delete,
						},
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{"nomos.dev"},
							APIVersions: []string{"v1"},
							Resources:   []string{"policynodes", "clusterpolicies"},
						},
					},
				},
				FailurePolicy: &failurePolicy,
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: controllerNamespace,
						Name:      controllerName,
					},
					CABundle: caCert,
				},
			},
		},
	}

	err = admissioncontroller.RegisterWebhook(clientset, webhookConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to self register webhook")
	}
	return nil
}

func main() {
	flag.Parse()
	log.Setup()

	glog.Info("Policy Admission Controller starting up")

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
		clientCert, err = admissioncontroller.GetAPIServerCert(clientset)
		if err != nil {
			glog.Fatal("Failed to get client cert: ", err)
		}
	}
	policyNodeInformer, err := admissioncontroller.SetupPolicyNodeInformer(config)
	if err != nil {
		glog.Fatal("Failed setting up policynode informer: ", err)
	}
	glog.Info("Waiting for informers to sync...")
	if !cache.WaitForCacheSync(nil, policyNodeInformer.Informer().HasSynced) {
		glog.Fatal("Failure while waiting for policynode informer to sync")
	}

	go service.ServeMetrics()

	server := service.Server(
		admissioncontroller.ServeFunc(
			policy.NewAdmitter(policyNodeInformer)),
		clientCert)

	stopChannel := make(chan struct{})
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		glog.Fatal("Failed to start https listener: ", err)
	}
	// nolint: errcheck
	defer listener.Close()

	go admissioncontroller.ServeTLS(server, listener, stopChannel)

	// Wait for endpoint to come up before self-registering
	err = admissioncontroller.WaitForEndpoint(
		clientset, controllerName, controllerNamespace, admissioncontroller.EndpointRegistrationTimeout)
	if err != nil {
		glog.Fatal("Failed waiting for endpoint: ", err)
	}

	// Finally register the webhook to block admission according to nomos policy constraints
	if err := selfRegister(clientset, *caBundleFile); err != nil {
		glog.Fatal("Failed to register webhook: ", err)
	}

	<-stopChannel
}
