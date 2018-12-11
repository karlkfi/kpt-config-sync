/*
Copyright 2018 Google LLC.

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

package apiobject

import (
	"context"
	"fmt"

	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Resource defines a collection of common k8s API object related
// functionalities that all bespin resources should implement.
type Resource interface {
	// GetParentReference returns the resource's parent reference.
	GetParentReference() bespinv1.ParentReference

	// GetID returns the resource ID from GCP. For example,
	// Folder.GetID() => GCP Folder ID "1234567";
	// Project.GetID() => GCP Project ID "myproject-1234".
	GetID() string

	// GetAnnotations returns the current annotations of the resource.
	GetAnnotations() map[string]string

	// SetAnnotations updates the annotations of the resource.
	SetAnnotations(map[string]string)
}

// Client talks to k8s API server.
type Client interface {
	Get(context.Context, types.NamespacedName, runtime.Object) error
}

// CheckAndSetParentAnnotation checks if the given resource's parent exists, and
// annotates the resource with its parent's ID. It returns an error if its parent
// is not fully created yet. A parent resource (can only be Folder or Organization)
// is created if it satisfies the following conditions:
// 1. it has been created in k8s API server - this can be determined by
//    Get() the resource by its name.
// 2. it has been created in the underlying platform (e.g. GCP) - this can be
//    determined by inspecting its Spec.ID field, which will be populated back
//    after the resource is successfully created in e.g. GCP.
func CheckAndSetParentAnnotation(ctx context.Context, c Client, rs Resource) error {
	// Empty parent reference meansthis is a top-level resource.
	parent := rs.GetParentReference()
	if (bespinv1.ParentReference{}) == parent {
		return nil
	}
	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: parent.Name}}
	switch parent.Kind {
	case bespinv1.OrganizationKind:
		org := &bespinv1.Organization{}
		err := c.Get(ctx, req.NamespacedName, org)
		if err != nil {
			return errors.Wrapf(err, "failed to get parent Organization instance: %v", req.NamespacedName)
		}
		annotate(rs, bespinv1.ParentOrganizationIDKey, org.GetID())
	case bespinv1.FolderKind:
		folder := &bespinv1.Folder{}
		err := c.Get(ctx, req.NamespacedName, folder)
		if err != nil {
			return errors.Wrapf(err, "failed to get parent Folder instance: %v", req.NamespacedName)
		}
		if folder.GetID() == "0" {
			return fmt.Errorf("%v parent Folder ID not specified (Folder maybe being created)", req.NamespacedName)
		}
		annotate(rs, bespinv1.ParentFolderIDKey, folder.GetID())
	default:
		return fmt.Errorf("invalid parent reference kind: %v", parent.Kind)
	}
	return nil
}

// annotate updates the resource's annotation map with the new annotation
// key and value.
func annotate(rs Resource, aKey, aValue string) {
	annotations := rs.GetAnnotations()
	annotations[aKey] = aValue
	rs.SetAnnotations(annotations)
}
