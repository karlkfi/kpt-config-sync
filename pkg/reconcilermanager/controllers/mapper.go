package controllers

import (
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// mapSecretToRepoSync return a mapping from the Secret object to the RepoSync
// object to reconcile.
// Return reconcile request with namespace parsed from the object name if the
// object is in `config-management-system namespace`.
func mapSecretToRepoSync() handler.ToRequestsFunc {
	return func(a handler.MapObject) []reconcile.Request {
		if a.Meta.GetNamespace() == configsync.ControllerNamespace {
			return reconcileRequest(a)
		}
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      v1alpha1.RepoSyncName,
					Namespace: a.Meta.GetNamespace(),
				},
			},
		}
	}
}

// mapObjectToRepoSync return reconcile request if the Object has owner RepoSync.
func mapObjectToRepoSync() handler.ToRequestsFunc {
	return func(a handler.MapObject) []reconcile.Request {
		return reconcileRequest(a)
	}
}

// reconcileRequest return reconcile request with namespace parsed from the
// Object name if the Object has owner RepoSync.
func reconcileRequest(a handler.MapObject) []reconcile.Request {
	for _, owner := range a.Meta.GetOwnerReferences() {
		if !(owner.Kind == configsync.RepoSyncKind && owner.APIVersion == v1alpha1.SchemeGroupVersion.String()) {
			continue
		}

		ns := parseRepoSyncReconciler(a.Meta.GetName(), a.Object)
		if ns == "" {
			continue
		}

		// Return request since we never have more than one ownerReference with
		// Kind RepoSync.
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      v1alpha1.RepoSyncName,
					Namespace: ns,
				},
			},
		}
	}
	return nil
}
