package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/reconcilermanager/controllers/secret"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// reconcilerBase provides common data and methods for the RepoSync and RootSync reconcilers
type reconcilerBase struct {
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

func (r *reconcilerBase) upsertConfigMaps(ctx context.Context, mutations []configMapMutation, refs []metav1.OwnerReference) ([]byte, error) {
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
			childCM.OwnerReferences = refs
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

func (r *reconcilerBase) upsertServiceAccount(ctx context.Context, name string, refs []metav1.OwnerReference) error {
	var childSA corev1.ServiceAccount
	childSA.Name = name
	childSA.Namespace = v1.NSConfigManagementSystem

	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childSA, func() error {
		childSA.OwnerReferences = refs
		return nil
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("ServiceAccount successfully reconciled", executedOperation, op)
	}
	return nil
}

func filterVolumes(existing []corev1.Volume, authType string, secretName string) []corev1.Volume {
	var updatedVolumes []corev1.Volume

	for _, volume := range existing {
		if volume.Name == gitCredentialVolume {
			// Don't mount git-creds volume if auth is 'none' or 'gcenode'
			if secret.SkipForAuth(authType) {
				continue
			}
			volume.Secret.SecretName = secretName
		}
		updatedVolumes = append(updatedVolumes, volume)
	}

	return updatedVolumes
}
