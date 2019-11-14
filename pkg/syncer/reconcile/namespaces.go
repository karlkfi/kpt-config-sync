package reconcile

import "strings"

const systemPrefix = "kube-"

// isSystemNamespace returns true if the namespace name denotes a Kubernetes system namespace
func isSystemNamespace(ns string) bool {
	return strings.HasPrefix(ns, systemPrefix)
}

// isManageableSystemNamespace returns true if ns is a system namespace that may be managed by Nomos.
func isManageableSystemNamespace(ns string) bool {
	// default is the "" namespace.
	// kube-system runs kubernetes system pods.
	// kube-public is a namespace created by kubeadm.
	return ns == "default" || ns == "kube-system" || ns == "kube-public"
}
