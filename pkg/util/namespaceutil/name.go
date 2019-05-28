package namespaceutil

import (
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	"k8s.io/apimachinery/pkg/util/validation"
)

var (
	// Namespaces that either exist on the kubernetes cluster by default or are reserved by Nomos.
	reservedNamespaces = map[string]bool{
		configmanagement.ControllerNamespace: true,
	}

	systemPrefix = "kube-"
)

// IsInvalid returns an error if the namespace name is reserved by the system
// or is not a valid RFC 1123 dns label.
func IsInvalid(name string) bool {
	ve := validation.IsDNS1123Label(name)
	return len(ve) != 0
}

// IsSystem returns true if the namespace name denotes a system namespace
func IsSystem(ns string) bool {
	return strings.HasPrefix(ns, systemPrefix)
}

// IsReserved returns true if the namespace is reserved.
func IsReserved(name string) bool {
	return reservedNamespaces[name]
}

// IsManageableSystem returns true if ns is a system namespace that may be
// managed by Nomos.
func IsManageableSystem(ns string) bool {
	// default is the "" namespace.
	// kube-system runs kubernetes system pods.
	// kube-public is a namespace created by kubeadm.
	return ns == "default" || ns == "kube-system" || ns == "kube-public"
}
