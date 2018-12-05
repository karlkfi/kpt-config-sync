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

// This package runs the hierarchical resource quota admission controller
package main

import (
	"flag"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/admissioncontroller/resourcequota"
	"github.com/google/nomos/pkg/generic-syncer/labeling"
	"github.com/google/nomos/pkg/service"
	"github.com/google/nomos/pkg/util/log"
	"github.com/pkg/errors"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	externalAdmissionHookConfigName = "resource-quota.nomos.dev"
	controllerNamespace             = "nomos-system"
	controllerName                  = "resourcequota-admission-controller"
)

var (
	caBundleFile = flag.String("ca-cert", "ca.crt", "Webhook server bundle cert used by api-server to authenticate the webhook server.")
	enablemTLS   = flag.Bool("enable-mutual-tls", false, "If set, enables mTLS verification of the client connecting to the admission controller.")
)

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

// register the webhook admission controller with the kube-apiserver.
func selfRegister(clientset *kubernetes.Clientset, caCertFile string) error {
	caCert, err := admissioncontroller.GetWebhookCert(caCertFile)
	if err != nil {
		return err
	}
	deployment, err := clientset.AppsV1().Deployments(controllerNamespace).Get(controllerName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "while obtaining current deployment")
	}
	gvk, err := apiutil.GVKForObject(deployment, scheme.Scheme)
	if err != nil {
		return err
	}

	failurePolicy := admissionregistrationv1beta1.Fail
	webhookConfig := &admissionregistrationv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   externalAdmissionHookConfigName,
			Labels: labeling.NomosSystem.New(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Name:       deployment.GetName(),
					UID:        deployment.GetUID(),
				},
			},
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name: externalAdmissionHookConfigName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update,
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{"*"},
						APIVersions: []string{"*"},
						// This list comes from the list of resource types Kubernetes ResourceQuota controller
						// handles. It can be derived from kubernetes/pkg/quota/evaluator/core/registry.go
						Resources: []string{"persistentvolumeclaims", "pods", "configmaps", "resourcequotas",
							"replicationcontrollers", "secrets", "services"},
					},
				}},
				FailurePolicy: &failurePolicy,
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: controllerNamespace,
						Name:      controllerName,
					},
					CABundle: caCert,
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						metav1.LabelSelectorRequirement{
							Key:      labeling.ResourceManagementKey,
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{labeling.Enabled},
						},
					},
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

	glog.Info("Hierarchical Resource Quota Admission Controller starting up")

	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to load in cluster config: %+v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client set: %+v", err)
	}
	var clientCert []byte
	if *enablemTLS {
		clientCert, err = admissioncontroller.GetAPIServerCert(clientset)
		if err != nil {
			glog.Fatalf("Failed to get client cert: %+v", err)
		}
	}
	policyNodeInformer, err := admissioncontroller.SetupPolicyNodeInformer(config)
	if err != nil {
		glog.Fatalf("Failed setting up policyNode informer: %+v", err)
	}
	resourceQuotaInformer, err := setupResourceQuotaInformer(config)
	if err != nil {
		glog.Fatalf("Failed setting up resourceQuota informer: %+v", err)
	}
	glog.Info("Waiting for informers to sync...")
	if !cache.WaitForCacheSync(nil, policyNodeInformer.Informer().HasSynced, resourceQuotaInformer.Informer().HasSynced) {
		glog.Fatal("Failure while waiting for informers to sync")
	}

	go service.ServeMetrics()

	server := service.Server(
		admissioncontroller.ServeFunc(
			resourcequota.NewAdmitter(policyNodeInformer, resourceQuotaInformer)),
		clientCert)

	stopChannel := make(chan struct{})
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		glog.Fatalf("Failed to start https listener: %+v", err)
	}
	// nolint: errcheck
	defer listener.Close()

	go admissioncontroller.ServeTLS(server, listener, stopChannel)

	// Wait for endpoint to come up before self-registering
	err = admissioncontroller.WaitForEndpoint(
		clientset, controllerName, controllerNamespace, admissioncontroller.EndpointRegistrationTimeout)
	if err != nil {
		glog.Fatalf("Failed waiting for endpoint: %+v", err)
	}

	// Finally register the webhook to block admission according to quota policy
	if err := selfRegister(clientset, *caBundleFile); err != nil {
		glog.Fatalf("Failed to register webhook: %+v", err)
	}

	<-stopChannel
}
