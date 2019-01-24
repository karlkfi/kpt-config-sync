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
	"time"

	"github.com/golang/glog"
	bespinv1 "github.com/google/nomos/pkg/api/policyascode/v1"
	"github.com/google/nomos/pkg/bespin-controllers/test/k8s"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// MaxRetries is the maximal number of retries for each Reconcile request.
	MaxRetries = 5

	// ReconcileTimeout is the timeout limit for each reconcile request.
	ReconcileTimeout = time.Minute * 5
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

// Update updates the object in k8s api server with the latest condition if the newobj needs
// to be updated. It returns true if the object doesn't need to be updated (thus the reconcile
// is essentially done), and returns non-nil error if there is a failure.
func Update(ctx context.Context, c client.Client, r record.EventRecorder, obj, newobj GenericObject) (bool, error) {
	if equality.Semantic.DeepEqual(obj, newobj) {
		glog.V(1).Infof("[%v] already update to date", obj.GetName())
		return true, nil
	}
	if err := setCondition(newobj); err != nil {
		return false, errors.Wrapf(err, "failed to set %v conditions", newobj.GetName())
	}
	if err := c.Update(ctx, newobj); err != nil {
		return false, errors.Wrapf(err, "failed to update %s in API server.", newobj.GetName())
	}
	r.Eventf(newobj, v1.EventTypeNormal, k8s.Updated, k8s.UpdatedMessage)
	return false, nil
}

func setCondition(obj GenericObject) error {
	conditions := []bespinv1.Condition{
		k8s.NewCustomReadyCondition(v1.ConditionTrue, k8s.Updated, k8s.UpdatedMessage),
	}
	switch objType := obj.(type) {
	case *bespinv1.Organization:
		objType.Status.Conditions = conditions
	case *bespinv1.Folder:
		objType.Status.Conditions = conditions
	case *bespinv1.Project:
		objType.Status.Conditions = conditions
	case *bespinv1.IAMPolicy:
		objType.Status.Conditions = conditions
	case *bespinv1.ClusterIAMPolicy:
		objType.Status.Conditions = conditions
	case *bespinv1.OrganizationPolicy:
		objType.Status.Conditions = conditions
	case *bespinv1.ClusterOrganizationPolicy:
		objType.Status.Conditions = conditions
	default:
		return fmt.Errorf("invalid resource type: %v", objType)
	}
	return nil
}
