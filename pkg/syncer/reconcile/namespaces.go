package reconcile

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// isManageableSystemNamespace returns true if ns is a system namespace that may be managed by Nomos.
func isManageableSystemNamespace(ns string) bool {
	// default is the "" namespace.
	// kube-system runs kubernetes system pods.
	// kube-public is a namespace created by kubeadm.
	// gatekeeper-system should never be deleted by ACM no matter how it was installed.
	return ns == metav1.NamespaceDefault || ns == metav1.NamespaceSystem || ns == metav1.NamespacePublic || ns == "gatekeeper-system"
}
