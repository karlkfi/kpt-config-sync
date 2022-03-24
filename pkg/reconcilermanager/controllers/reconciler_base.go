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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	v1 "kpt.dev/configsync/pkg/api/configmanagement/v1"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/api/configsync/v1beta1"
	hubv1 "kpt.dev/configsync/pkg/api/hub/v1"
	"kpt.dev/configsync/pkg/core"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/metrics"
	"kpt.dev/configsync/pkg/reconcilermanager"
	"kpt.dev/configsync/pkg/util"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// These are used as keys in calls to r.log.Info
	executedOperation    = "operation"
	operationSubjectName = "name"

	// gitSecretRefField is the path of the field in the RootSync|RepoSync CRDs
	// that we wish to use as the "object reference".
	// It will be used in both the indexing and watching.
	gitSecretRefField = ".spec.git.secretRef.name"

	// fleetMembershipName is the name of the fleet membership
	fleetMembershipName = "membership"
)

// reconcilerBase provides common data and methods for the RepoSync and RootSync reconcilers
type reconcilerBase struct {
	clusterName             string
	client                  client.Client
	log                     logr.Logger
	scheme                  *runtime.Scheme
	isAutopilotCluster      *bool
	reconcilerPollingPeriod time.Duration
	hydrationPollingPeriod  time.Duration
	membership              *hubv1.Membership
}

// configMapMutation provides an interface for named mutation functions passed to upsertConfigMaps
type configMapMutation struct {
	cmName string
	data   map[string]string
}

func (r *reconcilerBase) upsertConfigMaps(ctx context.Context, mutations []configMapMutation, labelMap map[string]string, refs ...metav1.OwnerReference) ([]byte, error) {
	configMapData := make(map[string]map[string]string)

	for _, mutation := range mutations {
		// CreateOrUpdate() takes a callback, “mutate”, which is where all changes to
		// the object must be performed.
		// The name and namespace  must be filled in prior to calling CreateOrUpdate()
		//
		// Under the hood, CreateOrUpdate() first calls Get() on the object. If the
		// object does not exist, Create() will be called. If it does exist, Update()
		// will be called. Just before calling either Create() or Update(), the mutate
		// callback will be called.

		// CreateOrUpdate configmaps for a root or namespace Reconciler.
		var childCM corev1.ConfigMap
		childCM.Name = mutation.cmName
		childCM.Namespace = v1.NSConfigManagementSystem
		op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCM, func() error {
			if len(refs) > 0 {
				childCM.OwnerReferences = refs
			}
			r.addLabels(&childCM, labelMap)
			childCM.Labels["app"] = reconcilermanager.Reconciler
			childCM.Data = mutation.data
			return nil
		})
		if err != nil {
			return nil, err
		}
		if op != controllerutil.OperationResultNone {
			r.log.Info("ConfigMap successfully reconciled", operationSubjectName, mutation.cmName, executedOperation, op)
		}

		configMapData[mutation.cmName] = mutation.data
	}

	// hash return 128 bit FNV-1 hash of data of the configmaps created by the controller.
	// Reconciler deployment's spec.template.annotation updated with the hash to recreate the
	// deployment in the event of configmap update.
	return hash(configMapData)
}

func (r *reconcilerBase) upsertServiceAccount(ctx context.Context, name, auth, email string, labelMap map[string]string, refs ...metav1.OwnerReference) error {
	var childSA corev1.ServiceAccount
	childSA.Name = name
	childSA.Namespace = v1.NSConfigManagementSystem
	r.addLabels(&childSA, labelMap)

	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childSA, func() error {
		// Update ownerRefs for RootSync ServiceAccount.
		// Do not set ownerRefs for RepoSync ServiceAccount, since Reconciler Manager,
		// performs garbage collection for Reposync controller resources.
		if len(refs) > 0 {
			childSA.OwnerReferences = refs
		}
		// Update annotation when Workload Identity is enabled on a GKE cluster.
		// In case, Workload Identity is not enabled on a cluster and spec.git.auth: gcpserviceaccount,
		// the added annotation will be a no-op.
		if auth == configsync.GitSecretGCPServiceAccount {
			core.SetAnnotation(&childSA, GCPSAAnnotationKey, email)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("ServiceAccount successfully reconciled", operationSubjectName, name, executedOperation, op)
	}
	return nil
}

type mutateFn func(client.Object) error

func (r *reconcilerBase) upsertDeployment(ctx context.Context, name, namespace string, labelMap map[string]string, mutateObject mutateFn) (controllerutil.OperationResult, error) {
	reconcilerDeployment := &appsv1.Deployment{}
	if err := parseDeployment(reconcilerDeployment); err != nil {
		return controllerutil.OperationResultNone, errors.Wrap(err, "failed to parse reconciler Deployment manifest from ConfigMap")
	}

	reconcilerDeployment.Name = name
	reconcilerDeployment.Namespace = namespace
	r.addLabels(reconcilerDeployment, labelMap)
	if err := mutateObject(reconcilerDeployment); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return r.createOrPatchDeployment(ctx, reconcilerDeployment)
}

// createOrPatchDeployment() first call Get() on the object. If the
// object does not exist, Create() will be called. If it does exist, Patch()
// will be called.
func (r *reconcilerBase) createOrPatchDeployment(ctx context.Context, obj *appsv1.Deployment) (controllerutil.OperationResult, error) {
	key := client.ObjectKeyFromObject(obj)

	existing := &appsv1.Deployment{}

	if err := r.client.Get(ctx, key, existing); err != nil {
		if !apierrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}
		r.log.Info("Resource not found, creating one", "Resource", obj.GetObjectKind().GroupVersionKind().Kind, "namespace/name", key.String())
		if err := r.client.Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultCreated, nil
	}

	// If Autopilot adjusts the resource requirements, use the current resource requirements.
	// Otherwise, use the resource requirements in the mutated deployment template.
	resourceRequirementChanged := false
	if r.isAutopilotCluster == nil {
		isAutopilot, err := util.IsGKEAutopilotCluster(r.client)
		if err != nil {
			r.log.Error(err, "unable to check if it is an Autopilot cluster")
			return controllerutil.OperationResultNone, err
		}
		r.isAutopilotCluster = &isAutopilot
	}
	if *r.isAutopilotCluster {
		for _, existingContainer := range existing.Spec.Template.Spec.Containers {
			for i, desiredContainer := range obj.Spec.Template.Spec.Containers {
				if existingContainer.Name == desiredContainer.Name &&
					!reflect.DeepEqual(obj.Spec.Template.Spec.Containers[i].Resources, existingContainer.Resources) {
					obj.Spec.Template.Spec.Containers[i].Resources = existingContainer.Resources
					resourceRequirementChanged = true
				}
			}
		}
		// Keep the autopilot annotation
		if obj.Annotations == nil {
			obj.Annotations = map[string]string{}
		}
		obj.Annotations[metadata.AutoPilotAnnotation] = core.GetAnnotation(existing, metadata.AutoPilotAnnotation)
	}
	if resourceRequirementChanged {
		r.log.V(3).Info("Container resource requirements diverged from the Deployment template because of the mutation made by the AutoPilot. The resource requirement override will be ignored.")
	}

	if reflect.DeepEqual(existing.Labels, obj.Labels) && reflect.DeepEqual(existing.Spec, obj.Spec) {
		return controllerutil.OperationResultNone, nil
	}

	r.log.Info("The Deployment needs to be updated", "name", obj.Name)
	if err := r.client.Update(ctx, obj); err != nil {
		// Let the next reconciliation retry the patch operation for valid request.
		if !apierrors.IsInvalid(err) {
			return controllerutil.OperationResultNone, err
		}
		// The provided data is invalid (e.g. http://b/196922619), so delete and re-create the resource.
		r.log.Error(err, "Failed to patch resource, deleting and re-creating the resource", "Resource", obj.GetObjectKind().GroupVersionKind().Kind, "namespace/name", key.String())
		if err := r.client.Delete(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		if err := r.client.Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	return controllerutil.OperationResultUpdated, nil
}

// deploymentStatus return standardized status for Deployment.
//
// For Deployments, we look at .status.conditions as well as the other properties
// under .status. Status will be Failed if the progress deadline has been exceeded.
// Code Reference: https://github.com/kubernetes-sigs/cli-utils/blob/v0.22.0/pkg/kstatus/status/core.go
// TODO (akulkapoor) Update to use the library kstatus once available.
func (r *reconcilerBase) deploymentStatus(ctx context.Context, key client.ObjectKey) (*deploymentStatus, error) {
	var depObj appsv1.Deployment
	if err := r.client.Get(ctx, key, &depObj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, errors.Errorf(
				"Deployment %s not found in namespace: %s.", key.Name, key.Namespace)
		}
		return nil, errors.Wrapf(err, "error while retrieving deployment")
	}
	return checkDeploymentConditions(&depObj)
}

func mutateContainerResource(ctx context.Context, c *corev1.Container, override v1beta1.OverrideSpec, reconcilerType string) {
	for _, override := range override.Resources {
		if override.ContainerName == c.Name {
			if !override.CPURequest.IsZero() {
				c.Resources.Requests[corev1.ResourceCPU] = override.CPURequest
				metrics.RecordResourceOverrideCount(ctx, reconcilerType, c.Name, "cpu")
			}
			if !override.CPULimit.IsZero() {
				c.Resources.Limits[corev1.ResourceCPU] = override.CPULimit
				metrics.RecordResourceOverrideCount(ctx, reconcilerType, c.Name, "cpu")
			}
			if !override.MemoryRequest.IsZero() {
				c.Resources.Requests[corev1.ResourceMemory] = override.MemoryRequest
				metrics.RecordResourceOverrideCount(ctx, reconcilerType, c.Name, "memory")
			}
			if !override.MemoryLimit.IsZero() {
				c.Resources.Limits[corev1.ResourceMemory] = override.MemoryLimit
				metrics.RecordResourceOverrideCount(ctx, reconcilerType, c.Name, "memory")
			}
		}
	}
}

// validateResourcesName will validate potential resource name using IsDNS1123Label function
// only configMap names are validated since generate the longest names compared to other resources
func (r *reconcilerBase) validateResourcesName(mutations []configMapMutation) error {
	for _, mutation := range mutations {
		name := mutation.cmName
		errs := validation.IsDNS1123Label(name)
		if len(errs) > 0 {
			return fmt.Errorf("The resource name %q is invalid: %s. To fix it, update the resource name", name, strings.Join(errs, ", "))
		}
	}
	return nil
}

// addLabels will copy the content of labelMaps to the current resource labels
func (r *reconcilerBase) addLabels(resource client.Object, labelMap map[string]string) {
	currentLabels := resource.GetLabels()
	if currentLabels == nil {
		currentLabels = make(map[string]string)
	}

	for key, value := range labelMap {
		currentLabels[key] = value
	}

	resource.SetLabels(currentLabels)

}

func (r *reconcilerBase) injectFleetWorkloadIdentityCredentials(podTemplate *corev1.PodTemplateSpec, gsaEmail string) error {
	content := map[string]interface{}{
		"type":                              "external_account",
		"audience":                          fmt.Sprintf("identitynamespace:%s:%s", r.membership.Spec.WorkloadIdentityPool, r.membership.Spec.IdentityProvider),
		"service_account_impersonation_url": fmt.Sprintf("https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken", gsaEmail),
		"subject_token_type":                "urn:ietf:params:oauth:token-type:jwt",
		"token_url":                         "https://sts.googleapis.com/v1/token",
		"credential_source": map[string]string{
			"file": filepath.Join(gcpKSATokenDir, gsaTokenPath),
		},
	}
	bytes, err := json.Marshal(content)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the Fleet Workload Identity credentials")
	}
	core.SetAnnotation(podTemplate, metadata.FleetWorkloadIdentityCredentials, string(bytes))
	return nil
}
