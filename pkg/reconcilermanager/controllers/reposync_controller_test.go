package controllers

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
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
	reposyncName         = "repo-sync"
	reposyncRepo         = "https://github.com/test/reposync/csp-config-management/"
	reposyncDir          = "foo-corp"
	reposyncSSHKey       = "ssh-key"

	unsupportedConfigMap = "xyz"
	unsupportedContainer = "abc"

	// Hash of all configmap.data created by Namespace Reconciler.
	nsAnnotation = "25f75852c888f43425978dc819f95ce9"
	// Updated hash of all configmap.data updated by Namespace Reconciler.
	nsUpdatedAnnotation = "2818908c6256a6bff5318ab1dbfe8fd5"
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

type configMapRef struct {
	name     string
	optional *bool
}

func gitSyncConfigMap(containerName string, configmap configMapRef) map[string][]configMapRef {
	result := make(map[string][]configMapRef)
	result[containerName] = []configMapRef{configmap}
	return result
}

func reconcilerDeploymentWithConfigMap() map[string][]configMapRef {
	return map[string][]configMapRef{
		reconciler: {
			{
				name:     reconciler,
				optional: pointer.BoolPtr(false),
			},
			{
				name:     SourceFormat,
				optional: pointer.BoolPtr(true),
			},
		},
		gitSync: {
			{
				name:     gitSync,
				optional: pointer.BoolPtr(false),
			},
		},
	}
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

// nsDeploymentWithEnvFrom returns appsv1.Deployment
// containerConfigMap contains map of container name and their respective configmaps.
func nsDeploymentWithEnvFrom(namespace, name string,
	containerConfigMap map[string][]configMapRef,
	annotation map[string]string,
	opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = namespace
	result.Name = buildRepoSyncName(name)
	result.Spec.Template.Annotations = annotation

	var container []corev1.Container
	for cntrName, cms := range containerConfigMap {
		cntr := fake.ContainerObject(cntrName)
		var eFromSource []corev1.EnvFromSource
		for _, cm := range cms {
			eFromSource = append(eFromSource, envFromSource(name, cm))
		}
		cntr.EnvFrom = append(cntr.EnvFrom, eFromSource...)
		container = append(container, *cntr)
	}
	result.Spec.Template.Spec = corev1.PodSpec{
		ServiceAccountName: buildRepoSyncName(name),
		Containers:         container,
	}
	return result
}

func envFromSource(name string, configMap configMapRef) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: buildRepoSyncName(name, configMap.name),
			},
			Optional: configMap.optional,
		},
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

	fakeClient := syncerFake.NewClient(t, s, objs...)
	testReconciler := NewRepoSyncReconciler(
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RepoSync"),
		s,
	)
	return fakeClient, testReconciler
}

func TestRepoSyncMutateConfigMap(t *testing.T) {
	testCases := []struct {
		name            string
		repoSync        *v1alpha1.RepoSync
		actualConfigMap *corev1.ConfigMap
		wantConfigMap   *corev1.ConfigMap
		wantErr         bool
	}{
		{
			name: "ConfigMap created",
			repoSync: repoSync(
				gitRevision,
				branch,
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, gitSync),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, gitSync),
				gitSyncData(gitRevision, branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap updated with revision number",
			repoSync: repoSync(
				gitUpdatedRevision,
				branch,
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, gitSync),
				gitSyncData(gitRevision, branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid)),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, gitSync),
				gitSyncData(gitUpdatedRevision, branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap mutate failed, Unsupported ConfigMap",
			repoSync: repoSync(
				gitRevision,
				branch,
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, unsupportedConfigMap),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, unsupportedConfigMap),
				gitSyncData(gitRevision, branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mutateRepoSyncConfigMap(tc.repoSync, tc.actualConfigMap)
			if tc.wantErr && err == nil {
				t.Errorf("mutateRepoSyncConfigMap() got error: %q, want error", err)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRepoSyncConfigMap() got error: %q, want error: nil", err)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.actualConfigMap, tc.wantConfigMap); diff != "" {
					t.Errorf("mutateRepoSyncConfigMap() got diff: %v\nwant: nil", diff)
				}
			}
		})
	}
}

func TestRepoSyncMutateDeployment(t *testing.T) {
	testCases := []struct {
		name             string
		repoSync         *v1alpha1.RepoSync
		actualDeployment *appsv1.Deployment
		wantDeployment   *appsv1.Deployment
		wantErr          bool
	}{
		{
			name: "Deployment created",
			repoSync: repoSync(
				gitRevision,
				branch,
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace),
				"git-sync"),
			wantDeployment: nsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				reposyncReqNamespace,
				gitSyncConfigMap(gitSync, configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				map[string]string{v1alpha1.ConfigMapAnnotationKey: "31323334"},
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: false,
		},
		{
			name: "Deployment failed, Unsupported container",
			repoSync: repoSync(
				gitRevision,
				branch,
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace),
				unsupportedContainer),
			wantDeployment: nsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				reposyncReqNamespace,
				gitSyncConfigMap(gitSync, configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				nil,
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
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

	rs := repoSync(gitRevision, branch, core.Name(reposyncName), core.Namespace(reposyncReqNamespace))
	reqNamespacedName := namespacedName(reposyncName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			buildRepoSyncName(reposyncReqNamespace, gitSync),
			gitSyncData(gitRevision, branch, reposyncRepo),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			buildRepoSyncName(reposyncReqNamespace, reconciler),
			reconcilerData(reposyncReqNamespace, reposyncDir, reposyncRepo, branch, gitRevision),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			buildRepoSyncName(reposyncReqNamespace, SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		buildRepoSyncName(reposyncReqNamespace),
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
	)

	wantDeployment := []*appsv1.Deployment{
		nsDeploymentWithEnvFrom(
			v1.NSConfigManagementSystem,
			reposyncReqNamespace,
			reconcilerDeploymentWithConfigMap(),
			nsDeploymentAnnotation(),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	// compare ServiceAccount.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantServiceAccount)], wantServiceAccount, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Service Account diff %s", diff)
	}

	// cmpDeployment compare ConfigMapRef field in containers.
	cmpDeployment(t, wantDeployment, fakeClient)
	t.Log("ConfigMap, Deployement and ServiceAccount successfully created")

	// Verify status updates.
	gotStatus := fakeClient.Objects[core.IDOf(rs)].(*v1alpha1.RepoSync).Status
	wantStatus := v1alpha1.RepoSyncStatus{
		SyncStatus: v1alpha1.SyncStatus{
			ObservedGeneration: rs.Generation,
			Reconciler:         buildRepoSyncName(reqNamespacedName.Namespace),
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
			buildRepoSyncName(reposyncReqNamespace, gitSync),
			gitSyncData(gitUpdatedRevision, branch, reposyncRepo),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			buildRepoSyncName(reposyncReqNamespace, reconciler),
			reconcilerData(reposyncReqNamespace, reposyncDir, reposyncRepo, branch, gitUpdatedRevision),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			buildRepoSyncName(reposyncReqNamespace, SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
	}

	wantDeployment = []*appsv1.Deployment{
		nsDeploymentWithEnvFrom(
			v1.NSConfigManagementSystem,
			reposyncReqNamespace,
			reconcilerDeploymentWithConfigMap(),
			nsDeploymentUpdatedAnnotation(),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	// cmpDeployment compare ConfigMapRef field in containers.
	cmpDeployment(t, wantDeployment, fakeClient)
	t.Log("ConfigMap and Deployement successfully updated")

	// Verify status updates.
	gotStatus = fakeClient.Objects[core.IDOf(rs)].(*v1alpha1.RepoSync).Status
	wantStatus.Conditions[0].Message = "Reconciler deployment was updated"
	if diff := cmp.Diff(wantStatus, gotStatus, ignoreTimes); diff != "" {
		t.Errorf("Status diff:\n%s", diff)
	}
}

func cmpDeployment(t *testing.T, want []*appsv1.Deployment, fakeClient *syncerFake.Client) {
	t.Helper()
	for _, de := range want {
		actual := fakeClient.Objects[core.IDOf(de)]
		a := actual.(*appsv1.Deployment)
		// Compare Annotations.
		if !reflect.DeepEqual(de.Spec.Template.Annotations, a.Spec.Template.Annotations) {
			t.Errorf("Unexpected Annotation found, got: %s,want: %s",
				a.Spec.Template.Annotations, de.Spec.Template.Annotations)
		}
		// Compare ServiceAccountName.
		if de.Spec.Template.Spec.ServiceAccountName != a.Spec.Template.Spec.ServiceAccountName {
			t.Errorf("Unexpected ServiceAccountName found,got: %s,want: %s",
				a.Spec.Template.Spec.ServiceAccountName, de.Spec.Template.Spec.ServiceAccountName)
		}
		// Compare Containers.
		for _, i := range de.Spec.Template.Spec.Containers {
			for _, j := range a.Spec.Template.Spec.Containers {
				if i.Name == j.Name {
					// Compare EnvFrom fields in the container.
					if diff := cmp.Diff(i.EnvFrom, j.EnvFrom,
						cmpopts.SortSlices(func(x, y corev1.EnvFromSource) bool { return x.ConfigMapRef.Name < y.ConfigMapRef.Name })); diff != "" {
						t.Errorf("Unexpected configMapRef found, diff %s", diff)
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
