package controllers

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	auth               = "ssh"
	branch             = "1.0.0"
	gitRevision        = "1.0.0.rc.8"
	gitUpdatedRevision = "1.1.0.rc.1"

	reposyncReqNamespace = "bookinfo"
	reposyncCRName       = "repo-sync"
	reposyncRepo         = "https://github.com/test/reposync/csp-config-management/"
	reposyncDir          = "foo-corp"
	reposyncSSHKey       = "ssh-key"
	reposyncCluster      = "abc-123"

	gcpSAEmail = "config-sync@cs-project.iam.gserviceaccount.com"

	pollingPeriod = "50ms"

	// Hash of all configmap.data created by Namespace Reconciler.
	nsAnnotation = "7c45f159e7c8005a792dcb402c078957"
	// Updated hash of all configmap.data updated by Namespace Reconciler.
	nsUpdatedAnnotation = "ad9eec9d09067c7aa5c339b3cef083f3"

	nsAnnotationGCENode        = "1e0a718052edc00039f6acc3738a02ae"
	nsUpdatedAnnotationGCENode = "4a5db0cdb29526ef77b8d3d9e3a18c06"
	nsAnnotationNone           = "16b1a82808b5bd13ed552122796c7ea4"

	nsDeploymentGCENodeChecksum        = "0347b9bd42dc798b1d2b2437f2babdf3"
	nsDeploymentSecretChecksum         = "526e033d0ad8c5fa617d2911b3ee7d8f"
	nsDeploymentSecretUpdatedChecksum  = "239b45bf27dea7b27d2128fd8467c856"
	nsDeploymentGCENodeUpdatedChecksum = "23f8caabbe7d083c1215ec1f0931eb29"
	nsDeploymentNoneChecksum           = "71d331f52d4a6fc3bb48cb034e443295"
)

// Set in init.
var filesystemPollingPeriod time.Duration

var parsedDeployment = func(de *appsv1.Deployment) error {
	de.TypeMeta = fake.ToTypeMeta(kinds.Deployment())
	de.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				constants.ReconcilerLabel: reconcilermanager.Reconciler,
			},
		},
		Replicas: &reconcilerDeploymentReplicaCount,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: defaultContainers(),
				Volumes:    deploymentSecretVolumes("git-creds"),
			},
		},
	}
	return nil
}

func init() {
	var err error
	filesystemPollingPeriod, err = time.ParseDuration(pollingPeriod)
	if err != nil {
		glog.Exitf("failed to parse polling period: %q, got error: %v, want error: nil", pollingPeriod, err)
	}
}

func reposyncRef(rev string) func(*v1alpha1.RepoSync) {
	return func(rs *v1alpha1.RepoSync) {
		rs.Spec.Revision = rev
	}
}

func reposyncBranch(branch string) func(*v1alpha1.RepoSync) {
	return func(rs *v1alpha1.RepoSync) {
		rs.Spec.Branch = branch
	}
}

func reposyncSecretType(auth string) func(*v1alpha1.RepoSync) {
	return func(rs *v1alpha1.RepoSync) {
		rs.Spec.Auth = auth
	}
}

func reposyncSecretRef(ref string) func(*v1alpha1.RepoSync) {
	return func(rs *v1alpha1.RepoSync) {
		rs.Spec.Git.SecretRef = v1alpha1.SecretReference{Name: ref}
	}
}

func reposyncGCPSAEmail(email string) func(sync *v1alpha1.RepoSync) {
	return func(sync *v1alpha1.RepoSync) {
		sync.Spec.GCPServiceAccountEmail = email
	}
}

func repoSync(opts ...func(*v1alpha1.RepoSync)) *v1alpha1.RepoSync {
	rs := fake.RepoSyncObject(core.Namespace(reposyncReqNamespace))
	rs.Spec.Repo = reposyncRepo
	rs.Spec.Dir = reposyncDir
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func rolebinding(name, namespace string, opts ...core.MetaMutator) *rbacv1.RoleBinding {
	result := fake.RoleBindingObject(opts...)
	result.Name = name

	result.RoleRef.Name = repoSyncPermissionsName()
	result.RoleRef.Kind = "ClusterRole"
	result.RoleRef.APIGroup = "rbac.authorization.k8s.io"

	var sub rbacv1.Subject
	sub.Kind = "ServiceAccount"
	sub.Name = reconciler.RepoSyncName(namespace)
	sub.Namespace = constants.ControllerNamespace
	result.Subjects = append(result.Subjects, sub)

	return result
}

func deploymentAnnotation(value string) map[string]string {
	return map[string]string{
		constants.ConfigMapAnnotationKey: value,
	}
}

func setupNSReconciler(t *testing.T, objs ...client.Object) (*syncerFake.Client, *RepoSyncReconciler) {
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
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantNamespaces := map[string]struct{}{
		reposyncReqNamespace: {},
	}

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
	)

	wantRoleBinding := rolebinding(
		repoSyncPermissionsName(),
		reposyncReqNamespace,
		core.Namespace(reposyncReqNamespace),
	)

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsDeploymentSecretChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap, ServiceAccount, RoleBinding, Service, and Deployment successfully created")

	// Test updating Configmaps and Deployment resources.
	rs.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMap = []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(options{
				ref:        gitUpdatedRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsDeploymentSecretUpdatedChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestRepoSyncAuthGCENode(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(constants.GitSecretGCENode))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs)
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantNamespaces := map[string]struct{}{
		reposyncReqNamespace: {},
	}

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsDeploymentGCENodeChecksum, nsReconcilerName),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")

	// Test updating Configmaps and Deployment resources.
	rs.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsDeploymentGCENodeUpdatedChecksum, nsReconcilerName),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRepoSyncAuthGCPServiceAccount(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(constants.GitSecretGCPServiceAccount), reposyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs)
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantNamespaces := map[string]struct{}{
		reposyncReqNamespace: {},
	}

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: constants.GitSecretGCPServiceAccount,
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(constants.GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsDeploymentGCENodeChecksum, nsReconcilerName),
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

	// Compare Deployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Resources successfully created")

	// Test updating Configmaps and Deployment resources.
	rs.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMap = []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(options{
				ref:        gitUpdatedRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: constants.GitSecretGCPServiceAccount,
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantServiceAccount = fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(constants.GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsDeploymentGCENodeUpdatedChecksum, nsReconcilerName),
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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Resources successfully updated")
}

func TestRepoSyncSwitchAuthTypes(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(constants.GitSecretGCPServiceAccount), reposyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources with GCPServiceAccount auth type.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantNamespaces := map[string]struct{}{
		reposyncReqNamespace: {},
	}

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: constants.GitSecretGCPServiceAccount,
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			repoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(constants.GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsDeploymentGCENodeChecksum, nsReconcilerName),
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

	// compare Deployment.
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Resources successfully created")

	// Test updating RepoSync resources with SSH auth type.
	rs.Spec.Auth = secretAuth
	rs.Spec.Git.SecretRef.Name = reposyncSSHKey
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsDeploymentSecretChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test updating RepoSync resources with None auth type.
	rs.Spec.Auth = noneAuth
	rs.Spec.SecretRef = v1alpha1.SecretReference{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotationNone)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			noneMutator(nsDeploymentNoneChecksum, nsReconcilerName),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRepoSyncReconcilerRestart(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsDeploymentSecretChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap, ServiceAccount, RoleBinding, Service, and Deployment successfully created")

	// Scale down the Reconciler Deployment to 0 replicas.
	deploymentCoreObject := fakeClient.Objects[core.IDOf(wantDeployments[0])]
	deployment := deploymentCoreObject.(*appsv1.Deployment)
	*deployment.Spec.Replicas = 0
	if err := fakeClient.Update(ctx, deployment); err != nil {
		t.Fatalf("failed to update the deployment request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

// validateDeployments validates that important fields in the `wants` deployments match those same fields in the deployments found in the fakeClient
func validateDeployments(wants []*appsv1.Deployment, fakeClient *syncerFake.Client) error {
	for _, want := range wants {
		gotCoreObject := fakeClient.Objects[core.IDOf(want)]
		got := gotCoreObject.(*appsv1.Deployment)

		// Compare Deployment Annotations
		if diff := cmp.Diff(want.Annotations, got.Annotations); diff != "" {
			return errors.Errorf("Unexpected Deployment Annotations found. Diff: %v", diff)
		}

		// Compare Deployment Template Annotations.
		if diff := cmp.Diff(want.Spec.Template.Annotations, got.Spec.Template.Annotations); diff != "" {
			return errors.Errorf("Unexpected Template Annotations found. Diff: %v", diff)
		}

		// Compare ServiceAccountName.
		if diff := cmp.Diff(want.Spec.Template.Spec.ServiceAccountName, got.Spec.Template.Spec.ServiceAccountName); diff != "" {
			return errors.Errorf("Unexpected ServiceAccountName. Diff: %v", diff)
		}

		// Compare Replicas
		if *want.Spec.Replicas != *got.Spec.Replicas {
			return errors.Errorf("Unexpected Replicas. want %d, got %d", *want.Spec.Replicas, *got.Spec.Replicas)
		}

		// Compare Containers.
		if len(want.Spec.Template.Spec.Containers) != len(got.Spec.Template.Spec.Containers) {
			return errors.Errorf("Unexpected Container count, want %d, got %d",
				len(want.Spec.Template.Spec.Containers), len(got.Spec.Template.Spec.Containers))
		}
		for _, i := range want.Spec.Template.Spec.Containers {
			for _, j := range got.Spec.Template.Spec.Containers {
				if i.Name == j.Name {
					// Compare EnvFrom fields in the container.
					if diff := cmp.Diff(i.EnvFrom, j.EnvFrom,
						cmpopts.SortSlices(func(x, y corev1.EnvFromSource) bool { return x.ConfigMapRef.Name < y.ConfigMapRef.Name })); diff != "" {
						return errors.Errorf("Unexpected configMapRef found, diff %s", diff)
					}
					// Compare VolumeMount fields in the container.
					if diff := cmp.Diff(i.VolumeMounts, j.VolumeMounts,
						cmpopts.SortSlices(func(x, y corev1.VolumeMount) bool { return x.Name < y.Name })); diff != "" {
						return errors.Errorf("Unexpected volumeMount found, diff %s", diff)
					}
				}
			}
		}

		// Compare Volumes
		if len(want.Spec.Template.Spec.Volumes) != len(got.Spec.Template.Spec.Volumes) {
			return errors.Errorf("Unexpected Volume count, want %d, got %d",
				len(want.Spec.Template.Spec.Volumes), len(got.Spec.Template.Spec.Volumes))
		}
		for _, wantVolume := range want.Spec.Template.Spec.Volumes {
			for _, gotVolume := range got.Spec.Template.Spec.Volumes {
				if wantVolume.Name == gotVolume.Name {
					// Compare VolumeSource
					if !reflect.DeepEqual(wantVolume.VolumeSource, gotVolume.VolumeSource) {
						return errors.Errorf("Unexpected volume source for volume %s, want %v, got %v",
							wantVolume.Name, wantVolume.VolumeSource, gotVolume.VolumeSource)
					}
				}
			}
		}
	}
	return nil
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
		core.Name(reconciler.RepoSyncName(rs.Namespace)),
	)
	var replicas int32 = 1
	dep.Spec.Replicas = &replicas
	for _, mut := range muts {
		mut(dep)
	}
	return dep
}
