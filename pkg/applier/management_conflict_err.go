// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package applier

import (
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ManagementConflictErrorCode is the error code for management conflict errors.
const ManagementConflictErrorCode = "1060"

var managementConflictErrorBuilder = status.NewErrorBuilder(ManagementConflictErrorCode)

// ManagementConflictError indicates that the passed resource is illegally
// declared in both the Root repository and a Namespace repository.
func ManagementConflictError(resource client.Object) status.Error {
	return managementConflictErrorBuilder.
		Sprintf("The %q reconciler cannot manage resources declared in the Root repository. "+
			"Remove the declaration for this resource from either the Namespace repository, or the Root repository.",
			resource.GetAnnotations()[metadata.ResourceManagerKey]).
		BuildWithResources(resource)
}
