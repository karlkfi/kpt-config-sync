package meta

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// Hardcoded list of GroupVersionKinds referenced when comparing resource types.
var (
	ClusterRole = rbacv1.SchemeGroupVersion.WithKind("ClusterRole")
	Namespace   = corev1.SchemeGroupVersion.WithKind("Namespace")
)
