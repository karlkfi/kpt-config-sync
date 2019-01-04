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

package veterrors

import (
	"fmt"

	"github.com/google/nomos/pkg/kinds"
	"github.com/pkg/errors"
)

// Codes for each Nomos error.
const (
	ReservedDirectoryNameErrorCode              = "1001"
	DuplicateDirectoryNameErrorCode             = "1002"
	IllegalNamespaceSubdirectoryErrorCode       = "1003"
	IllegalNamespaceSelectorAnnotationErrorCode = "1004"
	UnsyncableClusterObjectErrorCode            = "1005"
	UnsyncableNamespaceObjectErrorCode          = "1006"
	IllegalAbstractNamespaceObjectKindErrorCode = "1007"
	ConflictingResourceQuotaErrorCode           = "1008"
	IllegalNamespaceDeclarationErrorCode        = "1009"
	IllegalAnnotationDefinitionErrorCode        = "1010"
	IllegalLabelDefinitionErrorCode             = "1011"
	NamespaceSelectorMayNotHaveAnnotationCode   = "1012"
	ObjectHasUnknownClusterSelectorCode         = "1013"
	InvalidSelectorErrorCode                    = "1014" // TODO: Must refactor to use properly
	MissingDirectoryErrorCode                   = "1015"
	MissingRepoErrorCode                        = "1017"
	IllegalSubdirectoryErrorCode                = "1018"
	IllegalTopLevelNamespaceErrorCode           = "1019"
	InvalidNamespaceNameErrorCode               = "1020"
	UnknownObjectErrorCode                      = "1021" // Impossible to create consistent example.
	DuplicateSyncGroupKindErrorCode             = "1022"
	IllegalKindInSystemErrorCode                = "1024"
	MultipleRepoDefinitionsErrorCode            = "1025"
	MultipleConfigMapsErrorCode                 = "1026"
	UnsupportedRepoSpecVersionCode              = "1027"
	InvalidDirectoryNameErrorCode               = "1028"
	MetadataNameCollisionErrorCode              = "1029"
	MultipleNamespacesErrorCode                 = "1030"
	MissingObjectNameErrorCode                  = "1031"
	UnknownResourceInSyncErrorCode              = "1032"
	IllegalSystemResourcePlacementErrorCode     = "1033"
	UnsupportedResourceInSyncErrorCode          = "1034"
	IllegalHierarchyModeErrorCode               = "1035"
	InvalidMetadataNameErrorCode                = "1036"
	IllegalKindInClusterregistryErrorCode       = "1037"
	IllegalKindInNamespacesErrorCode            = "1038"
	UnknownResourceVersionInSyncErrorCode       = "1039"
	UndefinedErrorCode                          = "????"

	// Obsolete error codes. Do not reuse.
	//Unused1016 = "1016"
	//Unused1023 = "1023"
)

// Example returns a canonical example to use
func Example(code string) Error {
	switch code {
	case ReservedDirectoryNameErrorCode:
		return ReservedDirectoryNameError{Dir: "reserved"}
	case InvalidNamespaceNameErrorCode:
		return InvalidNamespaceNameError{ResourceID: &resourceID{source: "namespaces/foo/namespace.yaml", name: "bar", groupVersionKind: kinds.Namespace()}, Expected: "foo"}
	case UnknownResourceVersionInSyncErrorCode:
		return UnknownResourceVersionInSyncError{SyncID: &syncID{source: "system/rq-sync.yaml", groupVersionKind: kinds.ResourceQuota().GroupKind().WithVersion("v2")}}
	default:
		panic(errors.Errorf("programmer error: example undefined for %T", code))
	}
}

// Explanation returns documentation about what the bug is, why it occurs, and more information on
// how to fix it than just the error message.
func Explanation(code string) string {
	switch code {
	case ReservedDirectoryNameErrorCode:
		return `
GKE Policy Management defines several
[Reserved Namespaces](../management_flow.md#namespaces), and users may
[specify their own Reserved Namespaces](../system_config.md#reserved-namespaces).
Namespace and Abstract Namespace directories MUST NOT use these reserved names.
To fix:

1.  rename the directory,
1.  remove the directory, or
1.  remove the reserved namespace declaration.
`
	case InvalidNamespaceNameErrorCode:
		return `
A Namespace Resource MUST have a metadata.name that matches the name of its
directory. To fix, correct the offending Namespace's metadata.name or its
directory.
`
	default:
		panic(errors.Errorf("programmer error: explanation undefined for %T", code))
	}
}

// Error Defines a Kubernetes Nomos Vet error
// These are GKE Policy Management directory errors which are shown to the user and documented.
type Error interface {
	Error() string
	Code() string
}

// withPrefix formats the start of error messages consistently.
func format(err Error, format string, a ...interface{}) string {
	return fmt.Sprintf("KNV%s: ", err.Code()) + fmt.Sprintf(format, a...)
}
