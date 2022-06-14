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
	"sort"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/api/configsync/v1beta1"
	"kpt.dev/configsync/pkg/core"
	"kpt.dev/configsync/pkg/kinds"
	"kpt.dev/configsync/pkg/reconcilermanager"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *reconcilerBase) cleanup(ctx context.Context, reconcilerKey types.NamespacedName, gvk schema.GroupVersionKind) error {
	log := r.log.WithValues("object", reconcilerKey.String(), "kind", gvk.Kind)

	u := &unstructured.Unstructured{}
	u.SetName(reconcilerKey.Name)
	if reconcilerKey.Namespace != "" {
		u.SetNamespace(reconcilerKey.Namespace)
	}
	u.SetGroupVersionKind(gvk)
	err := r.client.Delete(ctx, u,
		client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.V(4).Info("Object already deleted")
			return nil
		}
	}
	log.Info("Object deleted successfully")
	return nil
}

func (r *RepoSyncReconciler) deleteSecret(ctx context.Context, reconcilerKey types.NamespacedName) error {
	secretList := &corev1.SecretList{}
	if err := r.client.List(ctx, secretList, client.InNamespace(reconcilerKey.Namespace)); err != nil {
		return err
	}

	for _, s := range secretList.Items {
		if strings.HasPrefix(s.Name, reconcilerKey.Name) {
			if err := r.cleanup(ctx, client.ObjectKeyFromObject(&s), kinds.Secret()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *reconcilerBase) deleteConfigmap(ctx context.Context, reconcilerKey types.NamespacedName) error {
	cms := []string{
		ReconcilerResourceName(reconcilerKey.Name, reconcilermanager.Reconciler),
		ReconcilerResourceName(reconcilerKey.Name, reconcilermanager.HydrationController),
		ReconcilerResourceName(reconcilerKey.Name, reconcilermanager.GitSync),
	}
	for _, c := range cms {
		key := types.NamespacedName{Namespace: reconcilerKey.Namespace, Name: c}
		if err := r.cleanup(ctx, key, kinds.ConfigMap()); err != nil {
			return err
		}
	}
	return nil
}

func (r *reconcilerBase) deleteServiceAccount(ctx context.Context, reconcilerKey types.NamespacedName) error {
	return r.cleanup(ctx, reconcilerKey, kinds.ServiceAccount())
}

func (r *RepoSyncReconciler) deleteRoleBinding(ctx context.Context, namespace string, reconcilerKey types.NamespacedName) error {
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      RepoSyncPermissionsName(),
	}
	log := r.log.WithValues("object", key.String(), "kind", kinds.RoleBinding())

	rb := &rbacv1.RoleBinding{}
	if err := r.client.Get(ctx, key, rb); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(4).Info("Object already deleted")
			return nil
		}
		return errors.Wrapf(err, "failed to get the RoleBinding object %s", key)
	}
	if !r.deleteRoleBindingSubject(rb, reconcilerKey) {
		log.V(4).Info("Subject already deleted")
		return nil
	}
	if len(rb.Subjects) == 0 {
		return r.cleanup(ctx, key, kinds.RoleBinding())
	}
	err := r.client.Update(ctx, rb)
	if err != nil {
		log.Info("Subject delete successful")
	}
	return err
}

func (r *reconcilerBase) deleteDeployment(ctx context.Context, reconcilerKey types.NamespacedName) error {
	return r.cleanup(ctx, reconcilerKey, kinds.Deployment())
}

func (r *RepoSyncReconciler) updateRoleBindingSubjects(rb *rbacv1.RoleBinding, rsList *v1beta1.RepoSyncList) {
	var subjects []rbacv1.Subject
	for _, rs := range rsList.Items {
		subjects = append(subjects, subject(core.NsReconcilerName(rs.Namespace, rs.Name),
			configsync.ControllerNamespace,
			"ServiceAccount"))
	}
	sort.SliceStable(subjects, func(i, j int) bool {
		return subjects[i].Name < subjects[j].Name
	})
	rb.Subjects = subjects
}

func (r *RepoSyncReconciler) deleteRoleBindingSubject(rb *rbacv1.RoleBinding, reconcilerKey types.NamespacedName) bool {
	subjectToDelete := subject(reconcilerKey.Name, reconcilerKey.Namespace, "ServiceAccount")
	found := false
	for i, subject := range rb.Subjects {
		if subject == subjectToDelete {
			found = true
			rb.Subjects = append(rb.Subjects[:i], rb.Subjects[i+1:]...)
			break
		}
	}
	return found
}

func (r *RootSyncReconciler) deleteClusterRoleBinding(ctx context.Context, reconcilerKey types.NamespacedName) error {
	key := client.ObjectKey{Name: RootSyncPermissionsName()}
	log := r.log.WithValues("object", key.String(), "kind", kinds.ClusterRoleBinding())

	crb := &rbacv1.ClusterRoleBinding{}
	if err := r.client.Get(ctx, key, crb); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(4).Info("Object already deleted")
			return nil
		}
		return errors.Wrapf(err, "failed to get the ClusterRoleBinding object %s", key)
	}
	if !r.deleteClusterRoleBindingSubject(crb, reconcilerKey) {
		log.V(4).Info("Subject already deleted")
		return nil
	}
	if len(crb.Subjects) == 0 {
		return r.cleanup(ctx, key, kinds.ClusterRoleBinding())
	}
	err := r.client.Update(ctx, crb)
	if err != nil {
		r.log.Info("Subject delete successful")
	}
	return err
}

func (r *RootSyncReconciler) deleteClusterRoleBindingSubject(crb *rbacv1.ClusterRoleBinding, reconcilerKey types.NamespacedName) bool {
	subjectToDelete := subject(reconcilerKey.Name, reconcilerKey.Namespace, "ServiceAccount")
	found := false
	for i, subject := range crb.Subjects {
		if subject == subjectToDelete {
			found = true
			crb.Subjects = append(crb.Subjects[:i], crb.Subjects[i+1:]...)
			break
		}
	}
	return found
}

func (r *RootSyncReconciler) updateClusterRoleBindingSubjects(crb *rbacv1.ClusterRoleBinding, rsList *v1beta1.RootSyncList) error {
	var subjects []rbacv1.Subject
	for _, rs := range rsList.Items {
		subjects = append(subjects, subject(core.RootReconcilerName(rs.Name),
			configsync.ControllerNamespace,
			"ServiceAccount"))
	}
	sort.SliceStable(subjects, func(i, j int) bool {
		return subjects[i].Name < subjects[j].Name
	})
	crb.Subjects = subjects
	return nil
}
