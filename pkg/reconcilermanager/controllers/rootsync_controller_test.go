package controllers

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	noneAuth             = "none"
	secretAuth           = "ssh"
	rootsyncReqNamespace = "config-management-system"
	rootsyncKind         = "RootSync"
	rootsyncName         = "root-sync"
	rootsyncRepo         = "https://github.com/test/rootsync/csp-config-management/"
	rootsyncDir          = "baz-corp"
	rootsyncCluster      = "abc-123"

	// Hash of all configmap.data created by Root Reconciler.
	rsAnnotation = "75a76c9c0de8e20c5e387df1a752f87f"
	// Updated hash of all configmap.data updated by Root Reconciler.
	rsUpdatedAnnotation = "afcc0fc36266b70500c33218f773bd7f"

	rsAnnotationGCENode        = "13c7343a532901cd51b815a9ff10db8c"
	rsUpdatedAnnotationGCENode = "87db1abb4c04ba6e9b0a4e7ba9423588"
	rsAnnotationNone           = "0acf0cf0d52e90d05476584c7eb5cdc0"

	rootsyncSSHKey = "root-ssh-key"

	deploymentGCENodeChecksum        = "5b0f1ba4ee9cb3a5f484fb6abfed194e"
	deploymentSecretChecksum         = "ceb0d9faa80cfc10463aaba067369f3b"
	deploymentSecretUpdatedChecksum  = "31f88adb18d1fc3a2bb62e22c8f525d1"
	deploymentGCENodeUpdatedChecksum = "86a3c5daa443b7e9efef2cf4f6332fb6"
	deploymentNoneChecksum           = "0f6f2d57c404b93c251f02f94b8b5f0c"
)

func clusterrolebinding(name string, opts ...core.MetaMutator) *rbacv1.ClusterRoleBinding {
	result := fake.ClusterRoleBindingObject(opts...)
	result.Name = name

	result.RoleRef.Name = "cluster-admin"
	result.RoleRef.Kind = "ClusterRole"
	result.RoleRef.APIGroup = "rbac.authorization.k8s.io"

	var sub rbacv1.Subject
	sub.Kind = "ServiceAccount"
	sub.Name = reconciler.RootSyncName
	sub.Namespace = constants.ControllerNamespace
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

func setupRootReconciler(t *testing.T, objs ...client.Object) (*syncerFake.Client, *RootSyncReconciler) {
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
		controllerruntime.Log.WithName("controllers").WithName("RootSync"),
		s,
	)
	return fakeClient, testReconciler
}

func rootsyncRef(rev string) func(*v1alpha1.RootSync) {
	return func(rs *v1alpha1.RootSync) {
		rs.Spec.Revision = rev
	}
}

func rootsyncBranch(branch string) func(*v1alpha1.RootSync) {
	return func(rs *v1alpha1.RootSync) {
		rs.Spec.Branch = branch
	}
}

func rootsyncSecretType(auth string) func(*v1alpha1.RootSync) {
	return func(rs *v1alpha1.RootSync) {
		rs.Spec.Auth = auth
	}
}

func rootsyncSecretRef(ref string) func(*v1alpha1.RootSync) {
	return func(rs *v1alpha1.RootSync) {
		rs.Spec.Git.SecretRef = v1alpha1.SecretReference{Name: ref}
	}
}

func rootsyncGCPSAEmail(email string) func(sync *v1alpha1.RootSync) {
	return func(sync *v1alpha1.RootSync) {
		sync.Spec.GCPServiceAccountEmail = email
	}
}

func rootSync(opts ...func(*v1alpha1.RootSync)) *v1alpha1.RootSync {
	rs := fake.RootSyncObject()
	rs.Spec.Repo = rootsyncRepo
	rs.Spec.Dir = rootsyncDir
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func TestRootSyncReconciler(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(constants.GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
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
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		reconciler.RootSyncName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(rootsyncKind, rootsyncName, ""),
		}),
	)

	wantClusterRoleBinding := clusterrolebinding(
		rootSyncPermissionsName(),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(rootsyncKind, rootsyncName, ""),
		}),
	)

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentSecretChecksum, reconciler.RootSyncName, rootsyncSSHKey),
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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap, ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Test updating Configmaps and Deployment resources.
	rs.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
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
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentSecretUpdatedChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestRootSyncAuthGCENode(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(constants.GitSecretGCENode))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotationGCENode)),
			setServiceAccountName(reconciler.RootSyncName),
			gceNodeMutator(deploymentGCENodeChecksum, reconciler.RootSyncName),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")

	// Test updating Deployment resources.
	rs.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotationGCENode)),
			setServiceAccountName(reconciler.RootSyncName),
			gceNodeMutator(deploymentGCENodeUpdatedChecksum, reconciler.RootSyncName),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncAuthGCPServiceAccount(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(constants.GitSecretGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
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
				secretType: constants.GitSecretGCPServiceAccount,
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		reconciler.RootSyncName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(rootsyncKind, rootsyncName, ""),
		}),
		core.Annotation(constants.GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotationGCENode)),
			setServiceAccountName(reconciler.RootSyncName),
			gceNodeMutator(deploymentGCENodeChecksum, reconciler.RootSyncName),
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

	// Test updating RootSync resources.
	rs.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotationGCENode)),
			setServiceAccountName(reconciler.RootSyncName),
			gceNodeMutator(deploymentGCENodeUpdatedChecksum, reconciler.RootSyncName),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncSwitchAuthTypes(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(constants.GitSecretGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources with GCPServiceAccount auth type.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
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
				secretType: constants.GitSecretGCPServiceAccount,
				period:     constants.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			rootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		reconciler.RootSyncName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(rootsyncKind, rootsyncName, ""),
		}),
		core.Annotation(constants.GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotationGCENode)),
			setServiceAccountName(reconciler.RootSyncName),
			gceNodeMutator(deploymentGCENodeChecksum, reconciler.RootSyncName),
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

	// Test updating RootSync resources with SSH auth type.
	rs.Spec.Auth = secretAuth
	rs.Spec.Git.SecretRef.Name = rootsyncSSHKey
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentSecretChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test updating RootSync resources with None auth type.
	rs.Spec.Auth = noneAuth
	rs.Spec.SecretRef = v1alpha1.SecretReference{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotationNone)),
			setServiceAccountName(reconciler.RootSyncName),
			noneMutator(deploymentNoneChecksum, reconciler.RootSyncName),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncReconcilerRestart(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(constants.GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentSecretChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")

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

	// Verify the Reconciler Deployment is updated to 1 replicas.
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

type depMutator func(*appsv1.Deployment)

func rootSyncDeployment(muts ...depMutator) *appsv1.Deployment {
	dep := fake.DeploymentObject(
		core.Namespace(v1.NSConfigManagementSystem),
		core.Name(reconciler.RootSyncName),
	)
	var replicas int32 = 1
	dep.Spec.Replicas = &replicas
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

func secretMutator(deploymentConfigChecksum, reconcilerName, secretName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Annotations[deploymentConfigChecksumAnnotationKey] = deploymentConfigChecksum
		dep.Spec.Template.Spec.Volumes = deploymentSecretVolumes(secretName)
		dep.Spec.Template.Spec.Containers = secretMountContainers(reconcilerName)
	}
}

func gceNodeMutator(deploymentConfigChecksum, reconcilerName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Annotations[deploymentConfigChecksumAnnotationKey] = deploymentConfigChecksum
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "repo"}}
		dep.Spec.Template.Spec.Containers = gceNodeContainers(reconcilerName)
	}
}

func noneMutator(deploymentConfigChecksum, reconcilerName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Annotations[deploymentConfigChecksumAnnotationKey] = deploymentConfigChecksum
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "repo"}}
		dep.Spec.Template.Spec.Containers = noneContainers(reconcilerName)
	}
}

func setAnnotations(annotations map[string]string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Annotations = annotations
	}
}

func defaultContainers() []corev1.Container {
	return []corev1.Container{
		{Name: reconcilermanager.Reconciler},
		{Name: reconcilermanager.GitSync, VolumeMounts: []corev1.VolumeMount{
			{Name: "repo", MountPath: "/repo"},
			{Name: "git-creds", MountPath: "/etc/git-secret", ReadOnly: true},
		}},
	}
}

func secretMountContainers(reconcilerName string) []corev1.Container {
	return []corev1.Container{
		{
			Name:    reconcilermanager.Reconciler,
			EnvFrom: reconcilerContainerEnvFrom(reconcilerName),
		},
		{
			Name:    reconcilermanager.GitSync,
			EnvFrom: gitSyncContainerEnvFrom(reconcilerName),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
				{Name: "git-creds", MountPath: "/etc/git-secret", ReadOnly: true},
			},
		},
	}
}

func noneContainers(reconcilerName string) []corev1.Container {
	return []corev1.Container{
		{
			Name:    reconcilermanager.Reconciler,
			EnvFrom: reconcilerContainerEnvFrom(reconcilerName),
		},
		{
			Name:    reconcilermanager.GitSync,
			EnvFrom: gitSyncContainerEnvFrom(reconcilerName),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
			}},
	}
}

func gceNodeContainers(reconcilerName string) []corev1.Container {
	containers := noneContainers(reconcilerName)
	containers = append(containers, corev1.Container{Name: gceNodeAskpassSidecarName})
	return containers
}

func deploymentSecretVolumes(secretName string) []corev1.Volume {
	return []corev1.Volume{
		{Name: "repo"},
		{Name: "git-creds", VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		}},
	}
}

func reconcilerContainerEnvFrom(reconcilerName string) []corev1.EnvFromSource {
	optionalTrue := true
	optionalFalse := false
	envFromSources := []corev1.EnvFromSource{
		{ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: reconcilerName + "-reconciler"},
			Optional:             &optionalFalse,
		}},
	}

	if reconcilerName == reconciler.RootSyncName {
		envFromSources = append(envFromSources, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: reconcilerName + "-source-format"},
				Optional:             &optionalTrue,
			},
		})
	}
	return envFromSources
}

func gitSyncContainerEnvFrom(reconcilerName string) []corev1.EnvFromSource {
	optionalFalse := false
	return []corev1.EnvFromSource{
		{ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: reconcilerName + "-git-sync"},
			Optional:             &optionalFalse,
		}},
	}
}
