// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"strings"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kpt.dev/configsync/pkg/kinds"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// only Autopilot clusters have nodes with the prefix “gk3-“.
	autopilotPrefix = "gk3-"
)

// AutopilotManagedNamespaces tracks the namespaces that are managed by GKE autopilot.
// ACM should not mutate or create any resources in these namespaces.
var AutopilotManagedNamespaces = map[string]bool{
	// The kube-system namespace is managed by Autopilot, meaning that all resources in this namespace cannot be altered and new resources cannot be created.
	// https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-overview#managed_namespaces
	metav1.NamespaceSystem: true,
}

// AutopilotManagedKinds tracks the GVKs that are managed by GKE autopilot.
// ACM should not mutate resources with the same GVKs.
var AutopilotManagedKinds = []schema.GroupVersionKind{
	// Autopilot modifies mutating webhooks objects: http://cloud/kubernetes-engine/docs/concepts/autopilot-overview#webhooks_limitations
	admissionregistrationv1.SchemeGroupVersion.WithKind("MutatingWebhookConfiguration"),
	admissionregistrationv1.SchemeGroupVersion.WithKind("MutatingWebhookConfigurationList"),
}

// IsAutopilotManagedNamespace returns if the input object is a namespace managed by the Autopilot cluster.
func IsAutopilotManagedNamespace(o client.Object) bool {
	if o.GetObjectKind().GroupVersionKind().GroupKind() != kinds.Namespace().GroupKind() {
		return false
	}
	return AutopilotManagedNamespaces[o.GetName()]
}

// IsGKEAutopilotCluster returns if the cluster is an autopilot cluster.
// Currently, only Autopilot clusters have node with the prefix `gk3-`, so we
// can use the node prefix to check the cluster type.
// GKE Autopilot scales to zero nodes since 1.21 when there is no user workloads.
// In the case of zero node, check the existence of the
// policycontrollerv2.config.common-webhooks.networking.gke.io validatingWebhookConfiguration.
// It exists on Autopilot clusters after GKE 1.20.
func IsGKEAutopilotCluster(c client.Client) (bool, error) {
	nodes := &corev1.NodeList{}
	if err := c.List(context.Background(), nodes); err == nil {
		for _, node := range nodes.Items {
			if strings.HasPrefix(node.Name, autopilotPrefix) {
				return true, nil
			}
		}
	}

	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	objectKey := client.ObjectKey{Name: "policycontrollerv2.config.common-webhooks.networking.gke.io"}
	err := c.Get(context.Background(), objectKey, webhook)
	if err == nil {
		return true, nil
	} else if apierrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}
