package util

import (
	"context"
	"strings"

	"github.com/google/nomos/pkg/kinds"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
// Currently, only Autopilot clusters have node with the prefix `gk3-`.
func IsGKEAutopilotCluster(c client.Client) (bool, error) {
	nodes := &corev1.NodeList{}
	if err := c.List(context.Background(), nodes); err != nil {
		return false, err
	}
	for _, node := range nodes.Items {
		if strings.HasPrefix(node.Name, autopilotPrefix) {
			return true, nil
		}
	}
	return false, nil
}
