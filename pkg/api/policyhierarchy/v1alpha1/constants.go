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

package v1alpha1

// SyncState indicates the state of a sync for resources of a particular group and kind.
type SyncState string

const (
	// Syncing indicates these resources are being actively managed by Nomos.
	Syncing SyncState = "syncing"
	// Error means errors were encountered while attempting to enable syncing for these resources.
	Error SyncState = "error"
)

// SyncFinalizer is a finalizer handled by Syncer to ensure Sync deletions complete before Importer writes ClusterPolicy
// and PolicyNode resources.
const SyncFinalizer = "syncer.nomos.dev"

// HierarchyModeType defines hierarchical behavior for namespaced objects.
type HierarchyModeType string

const (
	// HierarchyModeHierarchicalQuota indicates special aggregation behavior for ResourceQuota. With
	// this option, the policy is copied to namespaces, but it is also left in the abstract namespace.
	// There, the ResourceQuotaAdmissionController uses the value to enforce the ResourceQuota in
	// aggregate for all descendent namespaces.
	//
	// This mode can only be used for ResourceQuota.
	HierarchyModeHierarchicalQuota = "hierarchicalQuota"
	// HierarchyModeInherit indicates that the resource can appear in abstract namespace directories
	// and will be inherited by any descendent namespaces. Without this value on the Sync, resources
	// must not appear in abstract namespaces.
	HierarchyModeInherit = "inherit"
	// HierarchyModeNone indicates that the resource cannot appear in abstract namespace directories.
	// For most resource types, this is the same as default, and it's not necessary to specify this
	// value. But RoleBinding and ResourceQuota have different default behaviors, and this value is
	// used to disable inheritance behaviors for those types.
	HierarchyModeNone = "none"
	// HierarchyModeDefault is the default value. Default behavior is type-specific.
	HierarchyModeDefault = ""
)
