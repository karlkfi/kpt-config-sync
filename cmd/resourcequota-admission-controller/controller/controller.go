// Package controller defines the hierarchical resource quota admission controller
package controller

import (
	"time"

	"github.com/google/nomos/pkg/admissioncontroller"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/pkg/errors"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	informerscorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	externalAdmissionHookConfigName = "resource-quota." + configmanagement.GroupName
	// ControllerNamespace is the namespace the ResourceQuota Admission Controller lives in.
	ControllerNamespace = configmanagement.ControllerNamespace
	// ControllerName is the name of the ResourceQuota Admission Controller object.
	ControllerName = "resourcequota-admission-controller"
)

// SetupResourceQuotaInformer initizlizes the ResourceQuotaInformer.
func SetupResourceQuotaInformer(config *rest.Config) (informerscorev1.ResourceQuotaInformer, error) {
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

// SelfRegister registers the webhook admission controller with the kube-apiserver.
func SelfRegister(clientset *kubernetes.Clientset, caCertFile string) error {
	caCert, err := admissioncontroller.GetWebhookCert(caCertFile)
	if err != nil {
		return err
	}
	deployment, err := clientset.AppsV1().Deployments(ControllerNamespace).Get(ControllerName, metav1.GetOptions{})
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
			Name: externalAdmissionHookConfigName,
			Labels: map[string]string{
				v1.ConfigManagementSystemKey: v1.ConfigManagementSystemValue,
			},
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
						Namespace: ControllerNamespace,
						Name:      ControllerName,
					},
					CABundle: caCert,
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      v1.ConfigManagementQuotaKey,
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{v1.ConfigManagementQuotaValue},
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
