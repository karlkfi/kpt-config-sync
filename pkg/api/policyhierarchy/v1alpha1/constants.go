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

// ReservedNamespacesConfigMapName is the name of the ConfigMap specifying reserved namespaces.
const ReservedNamespacesConfigMapName = "nomos-reserved-namespaces"

// NamespaceAttribute is an attribute defining how Nomos reacts to reserved namespaces.
type NamespaceAttribute string

const (
	// ReservedAttribute means that these namespaces will not be managed by Nomos.
	ReservedAttribute NamespaceAttribute = "reserved"
)

// SyncFinalizer is a finalizer handled by Syncer to ensure Sync deletions complete before Importer writes ClusterPolicy
// and PolicyNode resources.
const SyncFinalizer = "syncer.nomos.dev"
