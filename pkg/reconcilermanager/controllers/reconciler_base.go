package controllers

import (
	"context"

	"github.com/go-logr/logr"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/reconcilermanager/controllers/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// reconcilerBase provides common data and methods for the RepoSync and RootSync reconcilers
type reconcilerBase struct {
	client client.Client
	log    logr.Logger
	scheme *runtime.Scheme
}

// configMapMutator provides an interface for mutation functions passed to upsertConfigMap
type configMapMutator func(*corev1.ConfigMap) error

// upsertConfigMap mutates and upserts a ConfigMap (retrieved by name) in the
// config-management-system namespace.
func (r *reconcilerBase) upsertConfigMap(ctx context.Context, name string, cmMut configMapMutator) error {
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
	childCM.Name = name
	childCM.Namespace = v1.NSConfigManagementSystem
	op, err := controllerruntime.CreateOrUpdate(ctx, r.client, &childCM, func() error {
		return cmMut(&childCM)
	})
	if err != nil {
		return err
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("ConfigMap successfully reconciled", operationSubjectName, name, executedOperation, op)
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
