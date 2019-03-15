/*
Copyright 2017 The Nomos Authors.
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

package resourcequota

// NamespaceTypeLabel is the label key used for nomos quotas.
const NamespaceTypeLabel = "nomos-namespace-type"
const (
	// NamespaceTypeWorkload is the value used for workload namespaces
	NamespaceTypeWorkload = "workload"
)

// ResourceQuotaObjectName is the resource name for quotas set by nomos.  We only allow one resource
// quota per namespace, so we hardcode the resource name.
const ResourceQuotaObjectName = "config-management-resource-quota"

// ResourceQuotaHierarchyName is the resource name for HierarchichalQuota.
const ResourceQuotaHierarchyName = "nomos-quota-hierarchy"

// NomosQuotaLabels are the labels applied to a workload namespace's quota object
var NomosQuotaLabels = NewNomosQuotaLabels()

// NewNomosQuotaLabels returns a new map of nomos quota labels since NomosQuotaLabels is mutable.
func NewNomosQuotaLabels() map[string]string {
	return map[string]string{NamespaceTypeLabel: NamespaceTypeWorkload}
}
