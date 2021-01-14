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
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/reconcilermanager"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/json"
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
	rsAnnotation = "eddad408a2b3e94314f4d5b5a441d5fb"
	// Updated hash of all configmap.data updated by Root Reconciler.
	rsUpdatedAnnotation = "f5f9d5d58163aabf40245d14623565cb"

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
	sub.Name = reconcilermanager.RootSyncName
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

func service(opts ...core.MetaMutator) *corev1.Service {
	result := fake.ServiceObject(opts...)

	result.Spec.Selector = map[string]string{"app": reconcilermanager.Reconciler}
	result.Spec.Ports = []corev1.ServicePort{{Name: "metrics", Port: 8675, TargetPort: intstr.FromString("metrics-port")}}
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
		filesystemPollingPeriod,
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RepoSync"),
		s,
	)
	return fakeClient, testReconciler
}

func rootSync(ref, branch, secretType, secretRef string, opts ...core.MetaMutator) *v1alpha1.RootSync {
	result := fake.RootSyncObject(opts...)
	result.Spec.Git = v1alpha1.Git{
		Repo:      rootsyncRepo,
		Revision:  ref,
		Branch:    branch,
		Dir:       rootsyncDir,
		Auth:      secretType,
		SecretRef: v1alpha1.SecretReference{Name: secretRef},
	}
	return result
}

func TestRootSyncReconciler(t *testing.T) {
	// Mock out parseService for testing.
	parseService = func(se *corev1.Service) error {
		se.Spec = corev1.ServiceSpec{
			Selector: map[string]string{"app": reconcilermanager.Reconciler},
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
					reconcilermanager.Reconciler: reconcilermanager.Reconciler,
				},
			},
			Replicas: &reconcilerDeploymentReplicaCount,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: reconcilermanager.Reconciler},
						{Name: reconcilermanager.GitSync},
					},
				},
			},
		}
		return nil
	}

	rs := rootSync(gitRevision, branch, v1alpha1.GitSecretConfigKeySSH, rootsyncSSHKey, core.Name(rootsyncName), core.Namespace(rootsyncReqNamespace))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     v1alpha1.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		reconcilermanager.RootSyncName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
	)

	wantClusterRoleBinding := clusterrolebinding(
		rootSyncPermissionsName(),
		core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
	)

	wantService := service(
		core.Name(reconcilermanager.RootSyncName),
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
	)

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(rsDeploymentAnnotation()),
			setServiceAccountName(reconcilermanager.RootSyncName),
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

	// compare Service.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantService)], wantService, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Service diff %s", diff)
	}

	validateDeployments(t, wantDeployments, fakeClient)
	t.Log("ConfigMap, ServiceAccount, ClusterRoleBinding and Deployment successfully created")

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
			rootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(options{
				ref:        gitUpdatedRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     v1alpha1.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(rsDeploymentUpdatedAnnotation()),
			setServiceAccountName(reconcilermanager.RootSyncName),
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
		core.Name(reconcilermanager.RootSyncName),
	)
	for _, mut := range muts {
		mut(dep)
	}
	return dep
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
