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
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

var (
	// Namespaces that either exist on the kubernetes cluster by default or are reserved by Stolos.
	reservedNamespaces = map[string]bool{
		"default":       true,
		"stolos-system": true,
	}

	reservedPrefix = "kube-"
)

// IsReservedOrInvalidNamespace returns an error if the namespace name is reserved by the system
// or is not a valid RFC 1123 dns label.
func IsReservedOrInvalidNamespace(name string) error {
	if isReserved(name) {
		return errors.Errorf("namespace %q is reserved by the system", name)
	}

	if ve := validation.IsDNS1123Label(name); len(ve) != 0 {
		return errors.New(ve[0])
	}
	return nil
}

func isReserved(name string) bool {
	if reservedNamespaces[name] {
		return true
	}

	if strings.HasPrefix(name, reservedPrefix) {
		return true
	}

	return false
}
