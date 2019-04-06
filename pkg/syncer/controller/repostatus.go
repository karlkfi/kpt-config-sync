/*
Copyright 2018 The CSP Config Management Authors.
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

package controller

import (
	"context"

	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/syncer/client"
	genericreconcile "github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const repoStatusControllerName = "repo-status"

// AddRepoStatus adds RepoStatus sync controller to the Manager.
func AddRepoStatus(ctx context.Context, mgr manager.Manager) error {
	syncClient := client.New(mgr.GetClient())
	rsc, err := k8scontroller.New(repoStatusControllerName, mgr, k8scontroller.Options{
		Reconciler: genericreconcile.NewRepoStatus(ctx, syncClient, metav1.Now),
	})
	if err != nil {
		return errors.Wrap(err, "could not create RepoStatus controller")
	}

	configHandler := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(configMapper),
	}
	if err = rsc.Watch(&source.Kind{Type: &v1.NamespaceConfig{}}, configHandler); err != nil {
		return errors.Wrapf(err, "could not watch NamespaceConfigs in the %q controller", repoStatusControllerName)
	}
	if err = rsc.Watch(&source.Kind{Type: &v1.ClusterConfig{}}, configHandler); err != nil {
		return errors.Wrapf(err, "could not watch ClusterConfigs in the %q controller", repoStatusControllerName)
	}

	repoHandler := &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(repoMapper),
	}
	if err = rsc.Watch(&source.Kind{Type: &v1.Repo{}}, repoHandler); err != nil {
		return errors.Wrapf(err, "could not watch Repos in the %q controller", repoStatusControllerName)
	}

	return nil
}

// Maps all configs into a single request that the RepoStatus controller just treats as an
// "invalidate" signal for the entire RepoStatus.
func configMapper(_ handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name: "invalidate-configs",
		},
	}}
}

// Maps the repo into a separate request to avoid race conditions between the importer and syncer
// when they are both updating configs and RepoStatus at the same time.
func repoMapper(_ handler.MapObject) []reconcile.Request {
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{
			Name: "invalidate-repo-status",
		},
	}}
}
