package reconcile

// isManageableSystemNamespace returns true if ns is a system namespace that may be managed by Nomos.
func isManageableSystemNamespace(ns string) bool {
	// default is the "" namespace.
	// kube-system runs kubernetes system pods.
	// kube-public is a namespace created by kubeadm.
	return ns == "default" || ns == "kube-system" || ns == "kube-public"
}
