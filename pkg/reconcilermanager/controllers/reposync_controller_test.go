package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
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
	reposyncCluster      = "abc-123"

	pollingPeriod = "50ms"

	// Hash of all configmap.data created by Namespace Reconciler.
	nsAnnotation = "a4fe2c51bedbb0e502883b3847f4bec3"
	// Updated hash of all configmap.data updated by Namespace Reconciler.
	nsUpdatedAnnotation = "92cc88e614c4eba1ce6adc081b955edf"
)

// Set in init.
var filesystemPollingPeriod time.Duration

func init() {
	var err error
	filesystemPollingPeriod, err = time.ParseDuration(pollingPeriod)
	if err != nil {
		glog.Exitf("failed to parse polling period: %q, got error: %v, want error: nil", pollingPeriod, err)
	}
}

func repoSync(ref, branch, secretType, secretRef string, opts ...core.MetaMutator) *v1alpha1.RepoSync {
	result := fake.RepoSyncObject(opts...)
	result.Spec.Git = v1alpha1.Git{
		Repo:      reposyncRepo,
		Revision:  ref,
		Branch:    branch,
		Dir:       reposyncDir,
		Auth:      secretType,
		SecretRef: v1alpha1.SecretReference{Name: secretRef},
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
	sub.Name = RepoSyncName(namespace)
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
		reposyncCluster,
		filesystemPollingPeriod,
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RepoSync"),
		s,
	)
	return fakeClient, testReconciler
}

func TestRepoSyncReconciler(t *testing.T) {
	// Mock out parseService for testing.
	parseService = func(se *corev1.Service) error {
		se.Spec = corev1.ServiceSpec{
			Selector: map[string]string{"app": reconciler},
			Ports: []corev1.ServicePort{
				{Name: "metrics", Port: 8675, TargetPort: intstr.FromString("metrics-port")},
			},
		}
		return nil
	}

	// Mock out parseDeployment for testing.
	parseDeployment = func(de *appsv1.Deployment) error {
		de.Spec = appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					reconciler: reconciler,
				},
			},
			Replicas: &reconcilerDeploymentReplicaCount,
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

	rs := repoSync(gitRevision, branch, auth, reposyncSSHKey, core.Namespace(reposyncReqNamespace))
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
			gitSyncData(options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     v1alpha1.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		RepoSyncName(reposyncReqNamespace),
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
	)

	wantRoleBinding := rolebinding(
		repoSyncPermissionsName(),
		reposyncReqNamespace,
		core.Namespace(reposyncReqNamespace),
		core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
	)

	wantService := service(
		core.Name(RepoSyncName(rs.Namespace)),
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
	)

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(nsDeploymentAnnotation()),
			setServiceAccountName(RepoSyncName(rs.Namespace)),
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

	// compare Service.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantService)], wantService, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Service diff %s", diff)
	}

	validateDeployments(t, wantDeployments, fakeClient)
	t.Log("ConfigMap, ServiceAccount, RoleBinding, Service, and Deployment successfully created")

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
			gitSyncData(options{
				ref:        gitUpdatedRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     v1alpha1.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference(ownerReference(reposyncKind, reposyncCRName, "")),
		),
	}

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(nsDeploymentUpdatedAnnotation()),
			setServiceAccountName(RepoSyncName(rs.Namespace)),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	validateDeployments(t, wantDeployments, fakeClient)
	t.Log("ConfigMap and Deployement successfully updated")
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
	dep := fake.DeploymentObject(
		core.Namespace(v1.NSConfigManagementSystem),
		core.Name(RepoSyncName(rs.Namespace)),
	)
	for _, mut := range muts {
		mut(dep)
	}
	return dep
}
