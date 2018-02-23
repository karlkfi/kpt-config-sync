/*
Copyright 2017 The Stolos Authors.
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
	"regexp"

	"github.com/pkg/errors"
	core_v1 "k8s.io/api/core/v1"
)

var (
	// TODO(b/73786828): This doesn't match DNS label definition.
	// We should probably use a K8S package instead of rolling our own.
	namespaceRegexPattern = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	// Pattern from output returned by kubectl
	// Matches: "namespace", "namespace-42", "42-namespace----43".
	// Does not match: "-namespace", "namespace-", "намеспаце".
	namespaceRe = regexp.MustCompile(namespaceRegexPattern)

	// Namespaces that either exist on the kubernetes cluster by default or are reserved by Stolos.
	// TODO(b/73788007): We probably want to reserve any "kube-" prefix.
	reservedNamespaces = map[string]bool{
		"default":       true,
		"kube-public":   true,
		"kube-system":   true,
		"stolos-system": true,
	}
)

// IsReserved returns true if the namespace name is reserved by the system.
func IsReserved(namespace core_v1.Namespace) bool {
	return reservedNamespaces[namespace.Name]
}

// IsReserved returns true if the namespace name is reserved by the system.
func IsReservedOrInvalidNamespace(name string) error {
	if reservedNamespaces[name] {
		return errors.Errorf("namespace %q is reserved by the system", name)
	}
	if !namespaceRe.MatchString(name) {
		return errors.Errorf("namespace %q is not a valid Kubernetes name", name)
	}
	return nil
}

// SanitizeNamespace will convert the namespace name to lowercase and assert it matches the
// formatting rules for namespaces.
func SanitizeNamespace(ns string) string {
	if !namespaceRe.MatchString(ns) {
		panic(errors.Errorf("namespace %q does not satisfy valid namespace pattern %s", ns, namespaceRegexPattern))
	}
	return ns
}
