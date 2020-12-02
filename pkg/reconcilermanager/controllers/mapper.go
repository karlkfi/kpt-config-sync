package controllers

import (
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// mapSecretToRepoSync return a mapping from the Secret object in the event to
// the RepoSync object to reconcile.
func mapSecretToRepoSync() handler.ToRequestsFunc {
	return func(a handler.MapObject) []reconcile.Request {
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

// mapDeploymentToRepoSync return a mapping from the Deployment Object with owner
// RepoSync Kind to reconcile.Request with namespace parsed from the Deployment name.
func mapDeploymentToRepoSync() handler.ToRequestsFunc {
	return func(a handler.MapObject) []reconcile.Request {
		for _, owner := range a.Meta.GetOwnerReferences() {
			if !(owner.Kind == configsync.RepoSyncKind && owner.APIVersion == v1alpha1.SchemeGroupVersion.String()) {
				continue
			}

			ns := parseRepoSyncReconciler(a.Meta.GetName())
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
}
