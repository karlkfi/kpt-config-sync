package controllers

import (
	"reflect"
	"strings"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/reconciler"

	"github.com/google/nomos/pkg/constants"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// mapSecretToRootSync define a mapping from the Secret object in the event to
// the RootSync object to reconcile.
//
// The Secret objects which should trigger a reconciliation of the RootSync object
// include `root-reconciler-token-...` and the Secret object specified in the
// `spec.git.secretRef` field of the RootSync object.
//
// The current implementation maps all the Secret objects without the `ns-reconciler-`
// prefix in the `config-management-system` namespace to the RootSync object.
func mapSecretToRootSync() handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		if a.GetNamespace() != constants.ControllerNamespace {
			return nil
		}

		if strings.HasPrefix(a.GetName(), reconciler.RepoSyncPrefix+"-") {
			return nil
		}

		glog.Infof("Changes to the secret (name: %s, namespace: %s) triggers a reconciliation for the RootSync object", a.GetName(), a.GetNamespace())
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      constants.RootSyncName,
					Namespace: constants.ControllerNamespace,
				},
			},
		}
	}
}

// mapSecretToRepoSync returns a mapping from a Secret in either 'config-management-system'
// namespace or a user namespace to a RepoSync to be reconciled.
func mapSecretToRepoSync() handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		if a.GetNamespace() == constants.ControllerNamespace {
			return reconcileRequest(a)
		}
		glog.Infof("Changes to the secret (name: %s, namespace: %s) triggers a reconciliation for the RepoSync object in the same namespace", a.GetName(), a.GetNamespace())
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      constants.RepoSyncName,
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
		if a.GetNamespace() == constants.ControllerNamespace {
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
	glog.Infof("Changes to the %s object (name: %s, namespace: %s) triggers a reconciliation for the RepoSync object in the %s namespace",
		reflect.TypeOf(a), a.GetName(), a.GetNamespace(), ns)
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      constants.RepoSyncName,
				Namespace: ns,
			},
		},
	}
}
