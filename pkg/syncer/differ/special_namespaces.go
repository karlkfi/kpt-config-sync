package differ

import (
	"github.com/google/nomos/pkg/policycontroller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var isManageableSystemNamespace = map[string]bool{
	// default is the "" namespace.
	metav1.NamespaceDefault: true,
	// kube-system runs kubernetes system pods.
	metav1.NamespaceSystem: true,
	// kube-public is a namespace created by kubeadm.
	metav1.NamespacePublic: true,
	// gatekeeper-system should never be deleted by ACM no matter how it was installed.
	policycontroller.NamespaceSystem: true,
}
