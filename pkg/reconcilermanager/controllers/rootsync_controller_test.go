package controllers

import (
	"context"
	"testing"

	"github.com/google/nomos/pkg/api/configsync"
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
	rbacv1 "k8s.io/api/rbac/v1"
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

func clusterrolebinding(name string, opts ...core.MetaMutator) *rbacv1.ClusterRoleBinding {
	result := fake.ClusterRoleBindingObject(opts...)
	result.Name = name

	result.RoleRef.Name = "cluster-admin"
	result.RoleRef.Kind = "ClusterRole"
	result.RoleRef.APIGroup = "rbac.authorization.k8s.io"

	var sub rbacv1.Subject
	sub.Kind = "ServiceAccount"
	sub.Name = rootSyncReconcilerName
	sub.Namespace = configsync.ControllerNamespace
	result.Subjects = append(result.Subjects, sub)

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

func setupRootReconciler(t *testing.T, objs ...runtime.Object) (*syncerFake.Client, *RootSyncReconciler) {
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
	testReconciler := NewRootSyncReconciler(
		rootsyncCluster,
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RepoSync"),
		s,
	)
	return fakeClient, testReconciler
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

func TestRootSyncMutateDeployment(t *testing.T) {
	testCases := []struct {
		name     string
		actual   *appsv1.Deployment
		expected *appsv1.Deployment
		wantErr  bool
	}{
		{
			name: "Deployment created",
			actual: rootSyncDeployment(
				setContainers(fake.ContainerObject(gitSync)),
				setVolumes(gitSyncVolume("")),
			),
			expected: rootSyncDeployment(
				setContainers(rootGitSyncContainer()),
				setAnnotations(map[string]string{v1alpha1.ConfigMapAnnotationKey: "31323334"}),
				setServiceAccountName(rootSyncReconcilerName),
				setVolumes(gitSyncVolume(rootsyncSSHKey)),
			),
			wantErr: false,
		},
		{
			name: "Deployment failed, Unsupported container",
			actual: rootSyncDeployment(
				setContainers(fake.ContainerObject(unsupportedContainer)),
			),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			declared := tc.actual.DeepCopyObject().(*appsv1.Deployment)
			testRS := rootSync(
				"1.0.0",
				branch,
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			)
			err := mutateRootSyncDeployment(testRS, tc.actual, declared, []byte("1234"))
			if tc.wantErr && err == nil {
				t.Errorf("mutateRepoSyncDeployment() returned error: %q, want error", err)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRepoSyncDeployment() returned error: %q, want error: nil", err)
			}
			if !tc.wantErr {
				diff := cmp.Diff(tc.actual, tc.expected)
				if diff != "" {
					t.Errorf("Deployment diff: %v", diff)
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
			rootSyncResourceName(gitSync),
			gitSyncData(gitRevision, branch, rootsyncRepo),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconciler),
			rootReconcilerData(declared.RootReconciler, rootsyncDir, rootsyncCluster, rootsyncRepo, branch, gitRevision),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		rootSyncReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
	)

	wantClusterRoleBinding := clusterrolebinding(
		rootSyncPermissionsName(),
		core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
	)

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(rsDeploymentAnnotation()),
			setServiceAccountName(rootSyncReconcilerName),
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

	// compare RoleBinding.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantClusterRoleBinding)], wantClusterRoleBinding, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ClusterRoleBinding diff %s", diff)
	}

	validateDeployments(t, wantDeployments, fakeClient)
	t.Log("ConfigMap, ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Verify status updates.
	gotStatus := fakeClient.Objects[core.IDOf(rs)].(*v1alpha1.RootSync).Status
	wantStatus := v1alpha1.RootSyncStatus{
		SyncStatus: v1alpha1.SyncStatus{
			ObservedGeneration: rs.Generation,
			Reconciler:         rootSyncReconcilerName,
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
			rootSyncResourceName(gitSync),
			gitSyncData(gitUpdatedRevision, branch, rootsyncRepo),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconciler),
			rootReconcilerData(declared.RootReconciler, rootsyncDir, rootsyncCluster, rootsyncRepo, branch, gitUpdatedRevision),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(rsDeploymentUpdatedAnnotation()),
			setServiceAccountName(rootSyncReconcilerName),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	validateDeployments(t, wantDeployments, fakeClient)
	t.Log("ConfigMap and Deployement successfully updated")
}

type depMutator func(*appsv1.Deployment)

func rootSyncDeployment(muts ...depMutator) *appsv1.Deployment {
	dep := fake.DeploymentObject(
		core.Namespace(v1.NSConfigManagementSystem),
		core.Name(rootSyncReconcilerName),
		core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid)),
	)

	for _, mut := range muts {
		mut(dep)
	}

	return dep
}

func setContainers(conts ...*corev1.Container) depMutator {
	return func(dep *appsv1.Deployment) {
		var templateContainers []corev1.Container
		for _, cont := range conts {
			templateContainers = append(templateContainers, *cont)
		}
		dep.Spec.Template.Spec.Containers = templateContainers
	}
}

func setServiceAccountName(name string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.ServiceAccountName = name
	}
}

func setAnnotations(annotations map[string]string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Annotations = annotations
	}
}

func setVolumes(vols ...corev1.Volume) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = vols
	}
}

func gitSyncVolume(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: gitCredentialVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}

func mutatedGitSyncContainer(objRef corev1.LocalObjectReference) *corev1.Container {
	return &corev1.Container{
		Name: gitSync,
		EnvFrom: []corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: objRef,
					Optional:             pointer.BoolPtr(false),
				},
			},
		},
	}
}

func rootGitSyncContainer() *corev1.Container {
	return mutatedGitSyncContainer(corev1.LocalObjectReference{
		Name: rootSyncResourceName(gitSync),
	})
}
