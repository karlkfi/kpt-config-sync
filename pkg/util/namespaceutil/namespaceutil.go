/*
Copyright 2017 The Kubernetes Authors.
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
	// Should match all allowed Kubernetes namespace names.
	namespaceRegexPattern = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
	// Pattern from output returned by kubectl
	// Matches: "namespace", "namespace-42", "42-namespace----43".
	// Does not match: "-namespace", "namespace-", "намеспаце".
	namespaceRe = regexp.MustCompile(namespaceRegexPattern)

	// Namespaces that exist on the kubernetes cluster by default.
	reservedNamespaces = map[string]bool{
		"default":       true,
		"kube-public":   true,
		"kube-system":   true,
		"stolos-system": true,
	}
)

// IsReserved returns true if the namespace name is one of the default ones created by kubernetes.
func IsReserved(namespace core_v1.Namespace) bool {
	return reservedNamespaces[namespace.Name]
}

// SanitizeNamespace will convert the namespace name to lowercase and assert it matches the
// formatting rules for namespaces.
func SanitizeNamespace(ns string) string {
	if !namespaceRe.MatchString(ns) {
		panic(errors.Errorf("Namespace \"%s\" does not satisfy valid namespace pattern %s", ns, namespaceRegexPattern))
	}
	return ns
}
