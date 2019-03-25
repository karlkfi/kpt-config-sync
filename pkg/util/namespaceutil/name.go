/*
Copyright 2017 The CSP Config Management Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
