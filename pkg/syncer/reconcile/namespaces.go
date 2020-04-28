package reconcile

// isManageableSystemNamespace returns true if ns is a system namespace that may be managed by Nomos.
func isManageableSystemNamespace(ns string) bool {
	// default is the "" namespace.
	// kube-system runs kubernetes system pods.
	// kube-public is a namespace created by kubeadm.
	// gatekeeper-system should never be deleted by ACM no matter how it was installed.
	return ns == "default" || ns == "kube-system" || ns == "kube-public" || ns == "gatekeeper-system"
}
