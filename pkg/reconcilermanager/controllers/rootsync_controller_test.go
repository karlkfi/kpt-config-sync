package controllers

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/api/configsync/v1alpha1"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	secretAuth           = "ssh"
	rootsyncReqNamespace = "config-management-system"
	rootsyncKind         = "RootSync"
	rootsyncName         = "root-sync"
	rootsyncRepo         = "https://github.com/test/rootsync/csp-config-management/"
	rootsyncDir          = "baz-corp"
	rootsyncCluster      = "abc-123"

	// Hash of all configmap.data created by Root Reconciler.
	rsAnnotation = "a91d9249d38347b60946f37d74ba2bd1"
	// Updated hash of all configmap.data updated by Root Reconciler.
	rsUpdatedAnnotation = "7890479a192ca90f82457814368324c9"

	rootsyncSSHKey = "root-ssh-key"
)

func configMap(namespace, name string, opts ...core.MetaMutator) *corev1.ConfigMap {
	result := fake.ConfigMapObject(opts...)
	result.Namespace = namespace
	result.Name = name
	return result
}

func configMapWithData(namespace, name string, data map[string]string, opts ...core.MetaMutator) *corev1.ConfigMap {
	result := fake.ConfigMapObject(opts...)
	result.Namespace = namespace
	result.Name = name
	result.Data = data
	return result
}

func secretData(t *testing.T, auth string) map[string][]byte {
	t.Helper()
	key, err := json.Marshal("test-key")
	if err != nil {
		t.Fatalf("failed to marshal test key: %v", err)
	}
	return map[string][]byte{
		auth: key,
	}
}

func secretObj(t *testing.T, name, auth string, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, auth)
	return result
}

func deployment(namespace, name string, containerName string, opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = namespace
	result.Name = name
	result.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			*fake.ContainerObject(containerName),
		},
	}
	return result
}

func rsDeploymentAnnotation() map[string]string {
	return map[string]string{
		v1alpha1.ConfigMapAnnotationKey: rsAnnotation,
	}
}

func rsDeploymentUpdatedAnnotation() map[string]string {
	return map[string]string{
		v1alpha1.ConfigMapAnnotationKey: rsUpdatedAnnotation,
	}
}

// rsDeploymentWithEnvFrom returns appsv1.Deployment
// containerConfigMap contains map of container name and their respective configmaps.
func rsDeploymentWithEnvFrom(namespace, name string,
	containerConfigMap map[string][]configMapRef,
	annotation map[string]string,
	opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = namespace
	result.Name = name
	result.Spec.Template.Annotations = annotation

	result.Spec.Template.Spec = corev1.PodSpec{
		ServiceAccountName: buildRootSyncName(),
		Containers:         reconcilerContainer(name, containerConfigMap),
	}
	return result
}

func reconcilerContainer(name string, containerConfigMap map[string][]configMapRef) []corev1.Container {
	var container []corev1.Container
	for cntrName, cms := range containerConfigMap {
		cntr := fake.ContainerObject(cntrName)
		var eFromSource []corev1.EnvFromSource
		for _, cm := range cms {
			eFromSource = append(eFromSource, rsEnvFromSource(cm))
		}
		cntr.EnvFrom = append(cntr.EnvFrom, eFromSource...)
		container = append(container, *cntr)
	}
	return container
}

func setupRootReconciler(t *testing.T, objs ...runtime.Object) (*syncerFake.Client, *RootSyncReconciler) {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := appsv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}

	fakeClient := syncerFake.NewClient(t, s, objs...)
	testReconciler := NewRootSyncReconciler(
		rootsyncCluster,
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RepoSync"),
		s,
	)
	return fakeClient, testReconciler
}

func rsEnvFromSource(configMap configMapRef) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: buildRootSyncName(configMap.name),
			},
			Optional: configMap.optional,
		},
	}
}

func rootSync(ref, branch string, opts ...core.MetaMutator) *v1alpha1.RootSync {
	result := fake.RootSyncObject(opts...)
	result.Spec.Git = v1alpha1.Git{
		Repo:      rootsyncRepo,
		Revision:  ref,
		Branch:    branch,
		Dir:       rootsyncDir,
		Auth:      auth,
		SecretRef: v1alpha1.SecretReference{Name: rootsyncSSHKey},
	}
	return result
}

func TestRootSyncMutateConfigMap(t *testing.T) {
	testCases := []struct {
		name            string
		rootSync        *v1alpha1.RootSync
		actualConfigMap *corev1.ConfigMap
		wantConfigMap   *corev1.ConfigMap
		wantErr         bool
	}{
		{
			name: "ConfigMap created",
			rootSync: rootSync(
				gitRevision,
				branch,
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncData(gitRevision, branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap updated with revision number",
			rootSync: rootSync(
				gitUpdatedRevision,
				branch,
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncData(gitRevision, branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid)),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncData(gitUpdatedRevision, branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap mutate failed, Unsupported ConfigMap",
			rootSync: rootSync(
				gitRevision,
				branch,
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				unsupportedConfigMap,
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				unsupportedConfigMap,
				gitSyncData(gitRevision, branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: true,
		},
	}

	rsResource := rootSync(gitRevision, branch, core.Name(rootsyncName), core.Namespace(rootsyncReqNamespace))
	_, testReconciler := setupRootReconciler(t, rsResource)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := testReconciler.mutateRootSyncConfigMap(*tc.rootSync, tc.actualConfigMap)
			if tc.wantErr && err == nil {
				t.Errorf("mutateRootSyncConfigMap() got error: %q, want error", err)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRootSyncConfigMap() got error: %q, want error: nil", err)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.actualConfigMap, tc.wantConfigMap); diff != "" {
					t.Errorf("mutateRootSyncConfigMap() got diff: %v\nwant: nil", diff)
				}
			}
		})
	}
}

func TestRootSyncMutateDeployment(t *testing.T) {
	testCases := []struct {
		name             string
		rootSync         *v1alpha1.RootSync
		actualDeployment *appsv1.Deployment
		wantDeployment   *appsv1.Deployment
		wantErr          bool
	}{
		{
			name: "Deployment created",
			rootSync: rootSync(
				"1.0.0",
				branch,
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSync),
			wantDeployment: rsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncConfigMap(gitSync, configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				map[string]string{v1alpha1.ConfigMapAnnotationKey: "31323334"},
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: false,
		},
		{
			name: "Deployment failed, Unsupported container",
			rootSync: rootSync(
				"1.0.0",
				branch,
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				unsupportedContainer),
			wantDeployment: rsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				gitSync,
				gitSyncConfigMap(gitSync, configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				nil,
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			declared := tc.actualDeployment.DeepCopyObject().(*appsv1.Deployment)
			err := mutateRootSyncDeployment(tc.rootSync, tc.actualDeployment, declared, []byte("1234"))
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

func TestRootSyncReconciler(t *testing.T) {
	// Mock out parseDeployment for testing.
	rsParseDeployment = func(de *appsv1.Deployment) error {
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

	rs := rootSync(gitRevision, branch, core.Name(rootsyncName), core.Namespace(rootsyncReqNamespace))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(gitSync),
			gitSyncData(gitRevision, branch, rootsyncRepo),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(reconciler),
			rootReconcilerData(declared.RootReconciler, rootsyncDir, rootsyncCluster, rootsyncRepo, branch, gitRevision),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		buildRootSyncName(),
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
	)

	wantDeployment := []*appsv1.Deployment{
		rsDeploymentWithEnvFrom(
			rootsyncReqNamespace,
			"root-reconciler",
			reconcilerDeploymentWithConfigMap(),
			rsDeploymentAnnotation(),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
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
		t.Errorf("ServiceAccount diff %s", diff)
	}

	// Compare ConfigMapRef field in containers.
	cmpDeployment(t, wantDeployment, fakeClient)
	t.Log("ConfigMap, Deployement and ServiceAccount successfully created")

	// Verify status updates.
	gotStatus := fakeClient.Objects[core.IDOf(rs)].(*v1alpha1.RootSync).Status
	wantStatus := v1alpha1.RootSyncStatus{
		SyncStatus: v1alpha1.SyncStatus{
			ObservedGeneration: rs.Generation,
			Reconciler:         buildRootSyncName(),
		},
		Conditions: []v1alpha1.RootSyncCondition{
			{
				Type:    v1alpha1.RootSyncReconciling,
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
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMap = []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(gitSync),
			gitSyncData(gitUpdatedRevision, branch, rootsyncRepo),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(reconciler),
			rootReconcilerData(declared.RootReconciler, rootsyncDir, rootsyncCluster, rootsyncRepo, branch, gitUpdatedRevision),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantDeployment = []*appsv1.Deployment{
		rsDeploymentWithEnvFrom(
			v1.NSConfigManagementSystem,
			"root-reconciler",
			reconcilerDeploymentWithConfigMap(),
			rsDeploymentUpdatedAnnotation(),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	// Compare ConfigMapRef field in containers.
	cmpDeployment(t, wantDeployment, fakeClient)
	t.Log("ConfigMap and Deployement successfully updated")
}
