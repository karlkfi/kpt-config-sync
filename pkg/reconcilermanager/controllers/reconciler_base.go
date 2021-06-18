package controllers

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// These are used as keys in calls to r.log.Info
	executedOperation    = "operation"
	operationSubjectName = "name"
)

// reconcilerBase provides common data and methods for the RepoSync and RootSync reconcilers
type reconcilerBase struct {
	clusterName             string
	client                  client.Client
	log                     logr.Logger
	scheme                  *runtime.Scheme
	filesystemPollingPeriod time.Duration
}

// configMapMutation provides an interface for named mutation functions passed to upsertConfigMaps
type configMapMutation struct {
	cmName string
	data   map[string]string
}

func (r *reconcilerBase) upsertConfigMaps(ctx context.Context, mutations []configMapMutation, refs ...metav1.OwnerReference) ([]byte, error) {
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
	// deployment in the event of configmap update. More information: go/csmr-update-deployment.
	return hash(configMapData)
}

func (r *reconcilerBase) upsertServiceAccount(ctx context.Context, name, auth, email string, refs ...metav1.OwnerReference) error {
	var childSA corev1.ServiceAccount
	childSA.Name = name
	childSA.Namespace = v1.NSConfigManagementSystem

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
		if auth == constants.GitSecretGCPServiceAccount {
			core.SetAnnotation(&childSA, constants.GCPSAAnnotationKey, email)
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

func (r *reconcilerBase) upsertDeployment(ctx context.Context, name, namespace string, f mutateFn) (controllerutil.OperationResult, error) {
	var childDep appsv1.Deployment
	if err := parseDeployment(&childDep); err != nil {
		return controllerutil.OperationResultNone, errors.Wrap(err, "failed to parse Deployment manifest from ConfigMap")
	}

	childDep.Name = name
	childDep.Namespace = namespace
	return r.createOrPatchDeployment(ctx, &childDep, f)
}

// createOrPatchDeployment() first call Get() on the object. If the
// object does not exist, Create() will be called. If it does exist, Patch()
// will be called.
func (r *reconcilerBase) createOrPatchDeployment(ctx context.Context, obj *appsv1.Deployment, mutateObject mutateFn) (controllerutil.OperationResult, error) {
	key := client.ObjectKeyFromObject(obj)

	existing := &appsv1.Deployment{}

	if err := r.client.Get(ctx, key, existing); err != nil {
		if !apierrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}
		r.log.Info("Resource not found, creating one", "Resource", obj.GetObjectKind().GroupVersionKind().Kind, "namespace/name", key.String())
		if err := mutateObject(obj); err != nil {
			return controllerutil.OperationResultNone, err
		}

		if err := r.client.Create(ctx, obj); err != nil {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultCreated, nil
	}

	// If the existing Deployment and the new Deployment have different `deploymentConfigChecksumAnnotation` annotations,
	// we need to patch the Deployment definitely.
	// If the existing Deployment and the new Deployment have the same `deploymentConfigChecksumAnnotation` annotation,
	// we should only patch the Deployment when `mutateObject(obj)` changes the object.
	if core.GetAnnotation(existing, deploymentConfigChecksumAnnotationKey) == core.GetAnnotation(obj, deploymentConfigChecksumAnnotationKey) {
		// We want to preserve the replicas from the deployment object.
		replicas := obj.Spec.Replicas
		obj = existing.DeepCopy()
		obj.Spec.Replicas = replicas
	}

	patch := client.MergeFrom(existing)

	if err := mutateObject(obj); err != nil {
		return controllerutil.OperationResultNone, err
	}

	if reflect.DeepEqual(existing, obj) {
		return controllerutil.OperationResultNone, nil
	}

	r.log.Info("The Deployment needs to be patched", "name", obj.Name)

	if err := r.client.Patch(ctx, obj, patch); err != nil {
		return controllerutil.OperationResultNone, err
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
