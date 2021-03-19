package controllers

import (
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// mapSecretToRepoSync returns a mapping from a Secret in either 'config-management-system'
// namespace or a user namespace to a RepoSync to be reconciled.
func mapSecretToRepoSync() handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		if a.GetNamespace() == configsync.ControllerNamespace {
			return reconcileRequest(a)
		}
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      v1alpha1.RepoSyncName,
					Namespace: a.GetNamespace(),
				},
			},
		}
	}
}

// mapObjectToRepoSync returns a mapping from an Object in 'config-management-system'
// namespace to a RepoSync to be reconciled.
func mapObjectToRepoSync() handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		if a.GetNamespace() == configsync.ControllerNamespace {
			return reconcileRequest(a)
		}
		return nil
	}
}

// reconcileRequest return reconcile request with namespace parsed from the
// Object name.
func reconcileRequest(a client.Object) []reconcile.Request {
	ns := nsOfReconciler(a)
	if ns == "" {
		return nil
	}

	// Return request with the namespace parsed from resource name.
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      v1alpha1.RepoSyncName,
				Namespace: ns,
			},
		},
	}
}
