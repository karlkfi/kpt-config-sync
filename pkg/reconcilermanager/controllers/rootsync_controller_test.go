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
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	rsAnnotation      = "5009e76f3fcd754688552d5d21b25b69"
	rsProxyAnnotation = "1c408d92779e06c559bf152fde87d9b0"
	// Updated hash of all configmap.data updated by Root Reconciler.
	rsUpdatedAnnotation = "fdcefb720a69315220f234f1675f8219"

	rsUpdatedAnnotationOverrideGitSyncDepth     = "567ce7dfd9f82312dc182d8a47e60d11"
	rsUpdatedAnnotationOverrideGitSyncDepthZero = "c04e047037af017710405de2988ad782"

	rsUpdatedAnnotationNoSSLVerify = "964761cb87ca908c20eec42cd6438df8"

	rsAnnotationGCENode        = "078fb3d0f21b6627061d2693d70e6770"
	rsUpdatedAnnotationGCENode = "cebd25606cc952db52fea0c5e65cdc54"
	rsAnnotationNone           = "836c0021b1532c9c6f2f4d11f661c12e"

	rootsyncSSHKey = "root-ssh-key"

	deploymentGCENodeChecksum        = "fb7197b50397eafc27bb6dbb96f93651"
	deploymentSecretChecksum         = "bf642e8754657fdd8b808701045a66b4"
	deploymentProxyChecksum          = "50429932edd585a028d4eb65a763978e"
	deploymentSecretUpdatedChecksum  = "9788d38a8155aee817d8300c2809389e"
	deploymentGCENodeUpdatedChecksum = "5fb27a85ac8b7ce37045c129282e853f"
	deploymentNoneChecksum           = "1dfe0ad848df07553cb36efa2cde7f44"

	// Checksums of the Deployment whose container resource limits are updated
	deploymentResourceLimitsChecksum                         = "3eb139bd47e5d3176d960bee1e3bb191"
	deploymentReconcilerLimitsChecksum                       = "4308d19d32ef68c29939159c3f021bc3"
	deploymentGitSyncMemLimitsChecksum                       = "a77249892e77d1bc72f5297fd461ac6e"
	deploymentReconcilerCPULimitsAndGitSyncMemLimitsChecksum = "eec8fd9816f06a1d673e20289b8788b0"

	rsDeploymentSecretOverrideGitSyncDepthChecksum     = "0cc33d5906b8d128eae663cb5c43c8ef"
	rsDeploymentSecretOverrideGitSyncDepthZeroChecksum = "c9a8d788642aee80acc1b6a10e18c8c1"

	rsDeploymentSecretNoSSLVerifyChecksum = "2b0d71b390447edeaacfac5770edee32"
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

func secretObjWithProxy(t *testing.T, name, auth string, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, auth)
	m2 := secretData(t, "https_proxy")
	for k, v := range m2 {
		result.Data[k] = v
	}
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
		hydrationPollingPeriod,
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

func rootsyncOverrideResourceLimits(containers []v1alpha1.ContainerResourcesSpec) func(sync *v1alpha1.RootSync) {
	return func(sync *v1alpha1.RootSync) {
		sync.Spec.Override = v1alpha1.OverrideSpec{
			Resources: containers,
		}
	}
}

func rootsyncOverrideGitSyncDepth(depth int64) func(*v1alpha1.RootSync) {
	return func(rs *v1alpha1.RootSync) {
		rs.Spec.Override.GitSyncDepth = &depth
	}
}

func rootsyncNoSSLVerify() func(*v1alpha1.RootSync) {
	return func(rs *v1alpha1.RootSync) {
		rs.Spec.Git.NoSSLVerify = true
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

func TestRootSyncReconcilerCreateAndUpdateRootSyncWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	overrideReconcilerAndGitSyncResourceLimits := []v1alpha1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH),
		rootsyncSecretRef(rootsyncSSHKey), rootsyncOverrideResourceLimits(overrideReconcilerAndGitSyncResourceLimits))
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, declared.RootReconciler, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentResourceLimitsChecksum, reconciler.RootSyncName, rootsyncSSHKey),
			containerResourceLimitsMutator(overrideReconcilerAndGitSyncResourceLimits),
		),
	}

	// compare ConfigMaps.
	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test overriding the CPU limits of the reconciler container and the memory limits of the git-sync container
	overrideReconcilerCPULimitsAndGitSyncMemLimits := []v1alpha1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPULimit:      resource.MustParse("1.2"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPULimit:      resource.MustParse("0.8"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			MemoryLimit:   resource.MustParse("888Gi"),
		},
	}

	rs.Spec.Override = v1alpha1.OverrideSpec{
		Resources: overrideReconcilerCPULimitsAndGitSyncMemLimits,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentReconcilerCPULimitsAndGitSyncMemLimitsChecksum, reconciler.RootSyncName, rootsyncSSHKey),
			containerResourceLimitsMutator(overrideReconcilerCPULimitsAndGitSyncMemLimits),
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

	// Clear rs.Spec.Override
	rs.Spec.Override = v1alpha1.OverrideSpec{}

	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
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
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestRootSyncReconcilerUpdateRootSyncWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, declared.RootReconciler, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test overriding the CPU/memory limits of both the reconciler and git-sync container
	overrideReconcilerAndGitSyncResourceLimits := []v1alpha1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs.Spec.Override = v1alpha1.OverrideSpec{
		Resources: overrideReconcilerAndGitSyncResourceLimits,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentResourceLimitsChecksum, reconciler.RootSyncName, rootsyncSSHKey),
			containerResourceLimitsMutator(overrideReconcilerAndGitSyncResourceLimits),
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

	// Test overriding the CPU/memory limits of the reconciler container
	overrideReconcilerResourceLimits := []v1alpha1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPULimit:      resource.MustParse("2"),
			MemoryLimit:   resource.MustParse("2Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPULimit:      resource.MustParse("1.3"),
			MemoryLimit:   resource.MustParse("4Gi"),
		},
	}

	rs.Spec.Override = v1alpha1.OverrideSpec{
		Resources: overrideReconcilerResourceLimits,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentReconcilerLimitsChecksum, reconciler.RootSyncName, rootsyncSSHKey),
			containerResourceLimitsMutator(overrideReconcilerResourceLimits),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Test overriding the memory limits of the git-sync container
	overrideGitSyncMemLimits := []v1alpha1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.GitSync,
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs.Spec.Override = v1alpha1.OverrideSpec{
		Resources: overrideGitSyncMemLimits,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments = []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentGitSyncMemLimitsChecksum, reconciler.RootSyncName, rootsyncSSHKey),
			containerResourceLimitsMutator(overrideGitSyncMemLimits),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1alpha1.OverrideSpec{}

	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
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
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestRootSyncCreateWithNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey), rootsyncNoSSLVerify())
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMapOverrideGitSyncDepth := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:         gitRevision,
				branch:      branch,
				repo:        rootsyncRepo,
				secretType:  "ssh",
				period:      configsync.DefaultPeriodSecs,
				proxy:       rs.Spec.Proxy,
				noSSLVerify: rs.Spec.NoSSLVerify,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeploymentsOverrideGitSyncDepth := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotationNoSSLVerify)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(rsDeploymentSecretNoSSLVerifyChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapOverrideGitSyncDepth {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsOverrideGitSyncDepth, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRootSyncUpdateNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:         gitRevision,
				branch:      branch,
				repo:        rootsyncRepo,
				secretType:  "ssh",
				period:      configsync.DefaultPeriodSecs,
				proxy:       rs.Spec.Proxy,
				noSSLVerify: rs.Spec.NoSSLVerify,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Set rs.Spec.NoSSLVerify to false
	rs.Spec.NoSSLVerify = false
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare ConfigMaps.
	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("No need to update ConfigMap and Deployment")

	// Set rs.Spec.NoSSLVerify to true
	rs.Spec.NoSSLVerify = true
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMapNoSSLVerify := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:         gitRevision,
				branch:      branch,
				repo:        rootsyncRepo,
				secretType:  "ssh",
				period:      configsync.DefaultPeriodSecs,
				proxy:       rs.Spec.Proxy,
				noSSLVerify: rs.Spec.NoSSLVerify,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeploymentsNoSSLVerify := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotationNoSSLVerify)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(rsDeploymentSecretNoSSLVerifyChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapNoSSLVerify {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsNoSSLVerify, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Set rs.Spec.NoSSLVerify to false
	rs.Spec.NoSSLVerify = false
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare ConfigMaps.
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

func TestRootSyncCreateWithOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey), rootsyncOverrideGitSyncDepth(5))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMapOverrideGitSyncDepth := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
				depth:      rs.Spec.Override.GitSyncDepth,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeploymentsOverrideGitSyncDepth := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotationOverrideGitSyncDepth)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(rsDeploymentSecretOverrideGitSyncDepthChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapOverrideGitSyncDepth {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsOverrideGitSyncDepth, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRootSyncUpdateOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap, ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Test overriding the git sync depth to a positive value
	var depth int64 = 5
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMapOverrideGitSyncDepth := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
				depth:      rs.Spec.Override.GitSyncDepth,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeploymentsOverrideGitSyncDepth := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotationOverrideGitSyncDepth)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(rsDeploymentSecretOverrideGitSyncDepthChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapOverrideGitSyncDepth {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsOverrideGitSyncDepth, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Test overriding the git sync depth to 0
	depth = 0
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMapOverrideGitSyncDepthZero := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
				depth:      rs.Spec.Override.GitSyncDepth,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeploymentsOverrideGitSyncDepthZero := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsUpdatedAnnotationOverrideGitSyncDepthZero)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(rsDeploymentSecretOverrideGitSyncDepthZeroChecksum, reconciler.RootSyncName, rootsyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapOverrideGitSyncDepthZero {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsOverrideGitSyncDepthZero, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Set rs.Spec.Override.GitSyncDepth to nil.
	rs.Spec.Override.GitSyncDepth = nil
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare ConfigMaps.
	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1alpha1.OverrideSpec{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare ConfigMaps.
	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("No need to update ConfigMap and Deployment.")
}

func TestRootSyncReconciler(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, declared.RootReconciler, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitUpdatedRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, declared.RootReconciler, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
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

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretGCENode))
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

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: configsync.GitSecretGCPServiceAccount,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, declared.RootReconciler, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
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
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
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

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
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
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: configsync.GitSecretGCPServiceAccount,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, declared.RootReconciler, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
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
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
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

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
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

func TestRootSyncWithProxy(t *testing.T) {
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretCookieFile), rootsyncSecretRef(reposyncCookie))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObjWithProxy(t, reposyncCookie, "cookie_file", core.Namespace(rootsyncReqNamespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       rootsyncRepo,
				secretType: configsync.GitSecretCookieFile,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, declared.RootReconciler, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.Reconciler),
			reconcilerData(rootsyncCluster, declared.RootReconciler, &rs.Spec.Git, pollingPeriod),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
		configMapWithData(
			rootsyncReqNamespace,
			RootSyncResourceName(reconcilermanager.SourceFormat),
			sourceFormatData(""),
			core.OwnerReference([]metav1.OwnerReference{
				ownerReference(rootsyncKind, rootsyncName, ""),
			}),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		rootSyncDeployment(
			setAnnotations(deploymentAnnotation(rsProxyAnnotation)),
			setServiceAccountName(reconciler.RootSyncName),
			secretMutator(deploymentProxyChecksum, reconciler.RootSyncName, reposyncCookie),
			envVarMutator("HTTPS_PROXY", reposyncCookie),
		),
	}

	// compare ConfigMaps.
	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("ConfigMap diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
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

func envVarMutator(envName, secretName string) depMutator {
	return func(dep *appsv1.Deployment) {
		for i, con := range dep.Spec.Template.Spec.Containers {
			if con.Name == reconcilermanager.GitSync {
				dep.Spec.Template.Spec.Containers[i].Env = []corev1.EnvVar{
					{
						Name: envName,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secretName,
								},
								Key: "https_proxy",
							},
						},
					},
				}
			}
		}
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

func containerResourceLimitsMutator(overrides []v1alpha1.ContainerResourcesSpec) depMutator {
	return func(dep *appsv1.Deployment) {
		for _, container := range dep.Spec.Template.Spec.Containers {
			switch container.Name {
			case reconcilermanager.Reconciler, reconcilermanager.GitSync, reconcilermanager.HydrationController:
				for _, override := range overrides {
					if override.ContainerName == container.Name {
						mutateContainerResourceLimits(&container, override)
					}
				}
			}
		}
	}
}

func mutateContainerResourceLimits(container *corev1.Container, resourceLimits v1alpha1.ContainerResourcesSpec) {
	if !resourceLimits.CPULimit.IsZero() {
		container.Resources.Limits[corev1.ResourceCPU] = resourceLimits.CPULimit
	} else {
		container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse("100m")
	}

	if !resourceLimits.MemoryLimit.IsZero() {
		container.Resources.Limits[corev1.ResourceMemory] = resourceLimits.MemoryLimit
	} else {
		container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse("100Mi")
	}
}

func defaultResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
	}
}

func defaultContainers() []corev1.Container {
	return []corev1.Container{
		{
			Name:      reconcilermanager.Reconciler,
			Resources: defaultResourceRequirements(),
		},
		{
			Name:      reconcilermanager.HydrationController,
			Resources: defaultResourceRequirements(),
		},
		{
			Name:      reconcilermanager.GitSync,
			Resources: defaultResourceRequirements(),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
				{Name: "git-creds", MountPath: "/etc/git-secret", ReadOnly: true},
			}},
	}
}

func secretMountContainers(reconcilerName string) []corev1.Container {
	return []corev1.Container{
		{
			Name:      reconcilermanager.Reconciler,
			Resources: defaultResourceRequirements(),
			EnvFrom:   reconcilerContainerEnvFrom(reconcilerName),
		},
		{
			Name:      reconcilermanager.HydrationController,
			Resources: defaultResourceRequirements(),
			EnvFrom:   hydrationContainerEnvFrom(reconcilerName),
		},
		{
			Name:      reconcilermanager.GitSync,
			Resources: defaultResourceRequirements(),
			EnvFrom:   gitSyncContainerEnvFrom(reconcilerName),
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
			Name:      reconcilermanager.Reconciler,
			Resources: defaultResourceRequirements(),
			EnvFrom:   reconcilerContainerEnvFrom(reconcilerName),
		},
		{
			Name:      reconcilermanager.HydrationController,
			Resources: defaultResourceRequirements(),
			EnvFrom:   hydrationContainerEnvFrom(reconcilerName),
		},
		{
			Name:      reconcilermanager.GitSync,
			Resources: defaultResourceRequirements(),
			EnvFrom:   gitSyncContainerEnvFrom(reconcilerName),
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

func hydrationContainerEnvFrom(reconcilerName string) []corev1.EnvFromSource {
	optionalFalse := false
	return []corev1.EnvFromSource{
		{ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: reconcilerName + "-hydration-controller"},
			Optional:             &optionalFalse,
		}},
	}
}
