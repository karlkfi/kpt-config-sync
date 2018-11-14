/*
Copyright 2018 The Nomos Authors.

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

package reserved

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/util/namespaceutil"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Namespaces represents a list of namespaces that will not be managed by Nomos.
// This means that they are outside the policy hierarchy and will not be
// modified or removed by any Nomos controllers.
//
// The ConfigMap must only consist of keys that represent the names of the
// namespaces that Nomos should not manage. The values must always be reserved.
//
// e.g.
//
// kind: ConfigMap
// apiVersion: v1
// data:
//   search: reserved
//   backend: reserved
//   frontend: reserved
// metadata:
//   name: nomos-reserved-namespaces
type Namespaces struct {
	configMap *v1.ConfigMap
}

// validate validates that reserved namespaces are well formed.
func (n *Namespaces) validate() error {
	if n.configMap.Name != v1alpha1.ReservedNamespacesConfigMapName {
		return errors.Errorf("reserved namespace configmap name %q is invalid, it must be %q", n.configMap.Name,
			v1alpha1.ReservedNamespacesConfigMapName)
	}
	for name, attribute := range n.configMap.Data {
		if errorStrings := validation.IsQualifiedName(name); len(errorStrings) > 0 {
			return errors.Errorf("reserved namespace %q is invalid: %v", name, errorStrings)
		}
		if v1alpha1.NamespaceAttribute(attribute) != v1alpha1.ReservedAttribute {
			return errors.Errorf("reserved namespace %q attribute %q is invalid", name, attribute)
		}
	}
	return nil
}

// IsReserved returns false when the namespace is not part of the reserved
// namespaces.
func (n *Namespaces) IsReserved(namespaceName string) bool {
	attribute, ok := n.configMap.Data[namespaceName]
	return (ok && v1alpha1.NamespaceAttribute(attribute) == v1alpha1.ReservedAttribute) ||
		namespaceutil.IsReserved(namespaceName)
}

// List returns the names of namespaces with the specified attribute.
func (n *Namespaces) List(wantAttribute v1alpha1.NamespaceAttribute) []string {
	var namespaces []string
	for namespace, attribute := range n.configMap.Data {
		if v1alpha1.NamespaceAttribute(attribute) == wantAttribute {
			namespaces = append(namespaces, namespace)
		}
	}
	return namespaces
}

// From gets Namespaces from the ConfigMap stored at the root of the policy
// hierarchy source of truth.
func From(reserved *v1.ConfigMap) (*Namespaces, error) {
	if reserved == nil {
		return EmptyNamespaces(), nil
	}

	ns := &Namespaces{configMap: reserved}
	if err := ns.validate(); err != nil {
		return EmptyNamespaces(), err
	}

	return ns, nil
}

// EmptyNamespaces returns an empty Namespaces struct that contains no
// reserved namespace dta.
func EmptyNamespaces() *Namespaces {
	return &Namespaces{
		configMap: &v1.ConfigMap{
			Data: make(map[string]string),
		},
	}
}
