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

package resource_quota

// NamespaceTypeLabel is the label key used for stolos quotas.
const NamespaceTypeLabel = "stolos-namespace-type"
const (
	// NamespaceTypePolicy is the value used for policy namespaces
	NamespaceTypePolicy = "policyspace"
	// NamespaceTypeWorkload is the value used for workload namespaces
	NamespaceTypeWorkload = "workload"
)

// ResourceQuotaObjectName is the resource name for quotas set by stolos.  We only allow one resource
// quota per namespace, so we hardcode the resource name.
const ResourceQuotaObjectName = "stolos-resource-quota"

// StolosQuotaLabels are the labels applied to a workload namespace's quota object
var StolosQuotaLabels = map[string]string{NamespaceTypeLabel: NamespaceTypeWorkload}

// PolicySpaceQuotaLabels are the labels applied to a policyspace's quota object
var PolicySpaceQuotaLabels = map[string]string{NamespaceTypeLabel: NamespaceTypePolicy}
