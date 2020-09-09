package controllers

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconcilermanager/controllers/secret"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	uid                = types.UID("1234")
	auth               = "ssh"
	branch             = "1.0.0"
	gitRevision        = "1.0.0.rc.8"
	gitUpdatedRevision = "1.1.0.rc.1"

	reposyncReqNamespace = "bookinfo"
	reposyncKind         = "RepoSync"
	reposyncCRName       = "repo-sync"
	reposyncRepo         = "https://github.com/test/reposync/csp-config-management/"
	reposyncDir          = "foo-corp"
	reposyncSSHKey       = "ssh-key"

	unsupportedContainer = "abc"

	// Hash of all configmap.data created by Namespace Reconciler.
	nsAnnotation = "0aa0b2b4b9109fffd8500509329b70da"
	// Updated hash of all configmap.data updated by Namespace Reconciler.
	nsUpdatedAnnotation = "78d51d3ac613da3303edacd4640f8f06"
)

func repoSync(ref, branch string, opts ...core.MetaMutator) *v1alpha1.RepoSync {
	result := fake.RepoSyncObject(opts...)
	result.Spec.Git = v1alpha1.Git{
		Repo:      reposyncRepo,
		Revision:  ref,
		Branch:    branch,
		Dir:       reposyncDir,
		Auth:      auth,
		SecretRef: v1alpha1.SecretReference{Name: reposyncSSHKey},
	}
	return result
}

func rolebinding(name, namespace string, opts ...core.MetaMutator) *rbacv1.RoleBinding {
	result := fake.RoleBindingObject(opts...)
	result.Name = name

	result.RoleRef.Name = repoSyncPermissionsName()
	result.RoleRef.Kind = "ClusterRole"
	result.RoleRef.APIGroup = "rbac.authorization.k8s.io"

	var sub rbacv1.Subject
	sub.Kind = "ServiceAccount"
	sub.Name = repoSyncName(namespace)
	sub.Namespace = configsync.ControllerNamespace
	result.Subjects = append(result.Subjects, sub)

	return result
}

func nsDeploymentAnnotation() map[string]string {
	return map[string]string{
		v1alpha1.ConfigMapAnnotationKey: nsAnnotation,
	}
}

func nsDeploymentUpdatedAnnotation() map[string]string {
	return map[string]string{
		v1alpha1.ConfigMapAnnotationKey: nsUpdatedAnnotation,
	}
}

func setupNSReconciler(t *testing.T, objs ...runtime.Object) (*syncerFake.Client, *RepoSyncReconciler) {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := appsv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := rbacv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}

	fakeClient := syncerFake.NewClient(t, s, objs...)
	testReconciler := NewRepoSyncReconciler(
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RepoSync"),
		s,
	)
	return fakeClient, testReconciler
}

func TestRepoSyncMutateDeployment(t *testing.T) {
	rs := repoSync(
		gitRevision,
		branch,
		core.Name(reposyncCRName),
		core.Namespace(reposyncReqNamespace),
		core.UID(uid),
	)

	testCases := []struct {
		name             string
		repoSync         *v1alpha1.RepoSync
		actualDeployment *appsv1.Deployment
		wantDeployment   *appsv1.Deployment
		wantErr          bool
	}{
		{
			name:     "Deployment created",
			repoSync: rs,
			actualDeployment: repoSyncDeployment(
				rs,
				setContainers(fake.ContainerObject(gitSync)),
				setVolumes(gitSyncVolume("")),
			),
			wantDeployment: repoSyncDeployment(
				rs,
				setContainers(repoGitSyncContainer(rs)),
				setAnnotations(map[string]string{v1alpha1.ConfigMapAnnotationKey: "31323334"}),
				setServiceAccountName(repoSyncName(rs.Namespace)),
				setVolumes(gitSyncVolume(secret.RepoSyncSecretName(rs.Namespace, rs.Spec.SecretRef.Name))),
			),
			wantErr: false,
		},
		{
			name:     "Deployment failed, Unsupported container",
			repoSync: rs,
			actualDeployment: repoSyncDeployment(
				rs,
				setContainers(fake.ContainerObject(unsupportedContainer)),
			),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			declared := tc.actualDeployment.DeepCopyObject().(*appsv1.Deployment)
			err := mutateRepoSyncDeployment(tc.repoSync, tc.actualDeployment, declared, []byte("1234"))
			if tc.wantErr && err == nil {
				t.Errorf("mutateRepoSyncDeployment() got error: %q, want error", err)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRepoSyncDeployment() got error: %q, want error: nil", err)
			}
			if !tc.wantErr {
				diff := cmp.Diff(tc.actualDeployment, tc.wantDeployment)
				if diff != "" {
					t.Errorf("mutateRepoSyncDeployment() got diff: %v\nwant: nil", diff)
				}
			}
		})
	}
}

func TestRepoSyncReconciler(t *testing.T) {
	// Mock out parseDeployment for testing.
	nsParseDeployment = func(de *appsv1.Deployment) error {
		de.Spec = appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: reconciler},
						{Name: gitSync},
					},
				},
			},
		}
		return nil
	}

	rs := repoSync(gitRevision, branch, core.Name(reposyncCRName), core.Namespace(reposyncReqNamespace))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, gitSync),
			gitSyncData(gitRevision, branch, reposyncRepo),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconciler),
			reconcilerData(reposyncReqNamespace, reposyncDir, reposyncRepo, branch, gitRevision),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		repoSyncName(reposyncReqNamespace),
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
	)

	wantRoleBinding := rolebinding(
		repoSyncPermissionsName(),
		reposyncReqNamespace,
		core.Namespace(reposyncReqNamespace),
		core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
	)

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(nsDeploymentAnnotation()),
			setServiceAccountName(repoSyncName(rs.Namespace)),
		),
	}

	// compare ConfigMaps.
	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	// compare ServiceAccount.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantServiceAccount)], wantServiceAccount, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Service Account diff %s", diff)
	}

	// compare RoleBinding.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantRoleBinding)], wantRoleBinding, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("RoleBinding diff %s", diff)
	}

	validateDeployments(t, wantDeployments, fakeClient)
	t.Log("ConfigMap, ServiceAccount, RoleBinding and Deployment successfully created")

	// Verify status updates.
	gotStatus := fakeClient.Objects[core.IDOf(rs)].(*v1alpha1.RepoSync).Status
	wantStatus := v1alpha1.RepoSyncStatus{
		SyncStatus: v1alpha1.SyncStatus{
			ObservedGeneration: rs.Generation,
			Reconciler:         repoSyncName(reqNamespacedName.Namespace),
		},
		Conditions: []v1alpha1.RepoSyncCondition{
			{
				Type:    v1alpha1.RepoSyncReconciling,
				Status:  metav1.ConditionTrue,
				Reason:  "Deployment",
				Message: "Reconciler deployment was created",
			},
		},
	}
	ignoreTimes := cmpopts.IgnoreFields(wantStatus.Conditions[0], "LastTransitionTime", "LastUpdateTime")
	if diff := cmp.Diff(wantStatus, gotStatus, ignoreTimes); diff != "" {
		t.Errorf("Status diff:\n%s", diff)
	}

	// Test updating Configmaps and Deployment resources.
	rs.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(context.Background(), rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMap = []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, gitSync),
			gitSyncData(gitUpdatedRevision, branch, reposyncRepo),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconciler),
			reconcilerData(reposyncReqNamespace, reposyncDir, reposyncRepo, branch, gitUpdatedRevision),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
	}

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(nsDeploymentUpdatedAnnotation()),
			setServiceAccountName(repoSyncName(rs.Namespace)),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	validateDeployments(t, wantDeployments, fakeClient)
	t.Log("ConfigMap and Deployement successfully updated")

	// Verify status updates.
	gotStatus = fakeClient.Objects[core.IDOf(rs)].(*v1alpha1.RepoSync).Status
	wantStatus.Conditions[0].Message = "Reconciler deployment was updated"
	if diff := cmp.Diff(wantStatus, gotStatus, ignoreTimes); diff != "" {
		t.Errorf("Status diff:\n%s", diff)
	}
}

// validateDeployments validates that important fields in the `wants` deployments match those same fields in the deployments found in the fakeClient
func validateDeployments(t *testing.T, wants []*appsv1.Deployment, fakeClient *syncerFake.Client) {
	t.Helper()
	for _, want := range wants {
		gotCoreObject := fakeClient.Objects[core.IDOf(want)]
		got := gotCoreObject.(*appsv1.Deployment)

		// Compare Annotations.
		if diff := cmp.Diff(want.Spec.Template.Annotations, got.Spec.Template.Annotations); diff != "" {
			t.Errorf("Unexpected Annotations found. Diff: %v", diff)
		}

		// Compare ServiceAccountName.
		if diff := cmp.Diff(want.Spec.Template.Spec.ServiceAccountName, got.Spec.Template.Spec.ServiceAccountName); diff != "" {
			t.Errorf("Unexpected ServiceAccountName. Diff: %v", diff)
		}

		// Compare Containers.
		for _, i := range want.Spec.Template.Spec.Containers {
			for _, j := range got.Spec.Template.Spec.Containers {
				if i.Name == j.Name {
					// Compare EnvFrom fields in the container.
					if diff := cmp.Diff(i.EnvFrom, j.EnvFrom,
						cmpopts.SortSlices(func(x, y corev1.EnvFromSource) bool { return x.ConfigMapRef.Name < y.ConfigMapRef.Name })); diff != "" {
						t.Errorf("Unexpected configMapRef found, diff %s", diff)
					}
					// Compare VolumeMount fields in the container.
					if diff := cmp.Diff(i.VolumeMounts, j.VolumeMounts,
						cmpopts.SortSlices(func(x, y corev1.VolumeMount) bool { return x.Name < y.Name })); diff != "" {
						t.Errorf("Unexpected volumeMount found, diff %s", diff)
					}
				}
			}
		}
	}
}

func namespacedName(name, namespace string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func repoSyncDeployment(rs *v1alpha1.RepoSync, muts ...depMutator) *appsv1.Deployment {
	oRefs := ownerReference(
		rs.GroupVersionKind().Kind,
		rs.Name,
		rs.UID,
	)
	dep := fake.DeploymentObject(
		core.Namespace(v1.NSConfigManagementSystem),
		core.Name(repoSyncName(rs.Namespace)),
		core.OwnerReference(oRefs),
	)

	for _, mut := range muts {
		mut(dep)
	}

	return dep
}

func repoGitSyncContainer(rs *v1alpha1.RepoSync) *corev1.Container {
	return mutatedGitSyncContainer(corev1.LocalObjectReference{
		Name: repoSyncResourceName(rs.Namespace, gitSync),
	})
}
