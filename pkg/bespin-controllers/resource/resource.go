/*
Copyright 2019 Google LLC.

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

package resource

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenericObject is an interface that combines functionalities from both:
// runtime.Object - used when taking to k8s api server.
// metav1.Object - used when working with object metadata.
type GenericObject interface {
	runtime.Object
	metav1.Object
}

// Get returns the resource from api server.
func Get(ctx context.Context, c client.Client, Kind, Name, Namespace string) (GenericObject, error) {
	var res GenericObject
	resName := types.NamespacedName{Name: Name, Namespace: Namespace}
	switch Kind {
	case bespinv1.OrganizationKind:
		res = &bespinv1.Organization{}
	case bespinv1.FolderKind:
		res = &bespinv1.Folder{}
	case bespinv1.ProjectKind:
		res = &bespinv1.Project{}
	default:
		return nil, fmt.Errorf("invalid kind: %v", Kind)
	}
	if err := c.Get(ctx, resName, res); err != nil {
		return nil, errors.Wrapf(err, "failed to get resource: %v/%v within namespace: %v", Kind, Name, Namespace)
	}
	return res, nil
}

// Update updates the Bespin GenericObject in k8s API server if the new object is not identical
// to the existing object.
// Note: r.Update() will trigger another Reconcile(), we should't update the API server
// when there is nothing changed.
func Update(ctx context.Context, c client.Client, obj, newobj GenericObject) error {
	// If there's no diff, we don't need to Update().
	if equality.Semantic.DeepEqual(obj, newobj) {
		glog.V(1).Infof("[%v] already update to date", obj.GetName())
		return nil
	}
	if err := c.Update(ctx, newobj); err != nil {
		return errors.Wrapf(err, "failed to update %s in API server.", newobj.GetName())
	}
	return nil
}
