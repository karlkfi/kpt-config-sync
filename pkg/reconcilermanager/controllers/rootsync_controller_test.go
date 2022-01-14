package controllers

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	noneAuth     = "none"
	secretAuth   = "ssh"
	rootsyncName = "my-root-sync"
	rootsyncRepo = "https://github.com/test/rootsync/csp-config-management/"
	rootsyncDir  = "baz-corp"
	testCluster  = "abc-123"

	// Hash of all configmap.data created by Root Reconciler.
	rsAnnotation = "8ed3b22230fe92b4e4314adadc652147"

	rsUpdatedAnnotationOverrideGitSyncDepth     = "ee51c116e781ed6db0e61cd298cb6d3f"
	rsUpdatedAnnotationOverrideGitSyncDepthZero = "17ba946e16664778aa2cb1cbb700071c"

	rsUpdatedAnnotationNoSSLVerify = "4835bf2ac62c90c95478a1699dba353a"

	rsAnnotationGCENode = "31330f1b533e8c364533e74f88d60d0e"
	rsAnnotationNone    = "501c415d1449547a00a031df51185aa0"

	rootsyncSSHKey = "root-ssh-key"
)

var rootReconcilerName = reconciler.RootReconcilerName(rootsyncName)

func clusterrolebinding(name, reconcilerName string, opts ...core.MetaMutator) *rbacv1.ClusterRoleBinding {
	result := fake.ClusterRoleBindingObject(opts...)
	result.Name = name

	result.RoleRef.Name = "cluster-admin"
	result.RoleRef.Kind = "ClusterRole"
	result.RoleRef.APIGroup = "rbac.authorization.k8s.io"

	var sub rbacv1.Subject
	sub.Kind = "ServiceAccount"
	sub.Name = reconcilerName
	sub.Namespace = configsync.ControllerNamespace
	result.Subjects = append(result.Subjects, sub)

	return result
}

func configMapWithData(namespace, name string, data map[string]string, opts ...core.MetaMutator) *corev1.ConfigMap {
	baseOpts := []core.MetaMutator{
		core.Labels(map[string]string{
			"app": reconcilermanager.Reconciler,
		}),
	}
	opts = append(baseOpts, opts...)
	result := fake.ConfigMapObject(opts...)
	result.Namespace = namespace
	result.Name = name
	result.Data = data
	return result
}

func secretObj(t *testing.T, name, auth string, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, "test-key", auth)
	return result
}

func secretObjWithProxy(t *testing.T, name, auth string, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, "test-key", auth)
	m2 := secretData(t, "test-key", "https_proxy")
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
	if err := admissionregistrationv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}

	fakeClient := syncerFake.NewClient(t, s, objs...)
	testReconciler := NewRootSyncReconciler(
		testCluster,
		filesystemPollingPeriod,
		hydrationPollingPeriod,
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RootSync"),
		s,
	)
	return fakeClient, testReconciler
}

func rootsyncRef(rev string) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Revision = rev
	}
}

func rootsyncBranch(branch string) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Branch = branch
	}
}

func rootsyncSecretType(auth string) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Auth = auth
	}
}

func rootsyncSecretRef(ref string) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Git.SecretRef = v1beta1.SecretReference{Name: ref}
	}
}

func rootsyncGCPSAEmail(email string) func(sync *v1beta1.RootSync) {
	return func(sync *v1beta1.RootSync) {
		sync.Spec.GCPServiceAccountEmail = email
	}
}

func rootsyncOverrideResources(containers []v1beta1.ContainerResourcesSpec) func(sync *v1beta1.RootSync) {
	return func(sync *v1beta1.RootSync) {
		sync.Spec.Override = v1beta1.OverrideSpec{
			Resources: containers,
		}
	}
}

func rootsyncOverrideGitSyncDepth(depth int64) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Override.GitSyncDepth = &depth
	}
}

func rootsyncNoSSLVerify() func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Git.NoSSLVerify = true
	}
}

func rootSync(name string, opts ...func(*v1beta1.RootSync)) *v1beta1.RootSync {
	rs := fake.RootSyncObjectV1Beta1(core.Name(name))
	rs.Spec.Repo = rootsyncRepo
	rs.Spec.Dir = rootsyncDir
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func TestCreateAndUpdateRootReconcilerWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	overrideAllContainerResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH),
		rootsyncSecretRef(rootsyncSSHKey), rootsyncOverrideResources(overrideAllContainerResources))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
		containerResourcesMutator(overrideAllContainerResources),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test overriding the CPU resources of the reconciler and hydration-container and the memory resources of the git-sync container
	overrideSelectedResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("1"),
			CPULimit:      resource.MustParse("1.2"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPULimit:      resource.MustParse("0.8"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			MemoryRequest: resource.MustParse("800Gi"),
			MemoryLimit:   resource.MustParse("888Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideSelectedResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
		containerResourcesMutator(overrideSelectedResources),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}

	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestUpdateRootReconcilerWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test overriding the CPU/memory requests and limits of the reconciler, hydration-controller, and git-sync container
	overrideAllContainerResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideAllContainerResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
		containerResourcesMutator(overrideAllContainerResources),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test overriding the CPU/memory limits of the reconciler and hydration-controller containers
	overrideReconcilerAndHydrationResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("1"),
			CPULimit:      resource.MustParse("2"),
			MemoryRequest: resource.MustParse("1Gi"),
			MemoryLimit:   resource.MustParse("2Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("1.1"),
			CPULimit:      resource.MustParse("1.3"),
			MemoryRequest: resource.MustParse("3Gi"),
			MemoryLimit:   resource.MustParse("4Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerAndHydrationResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
		containerResourcesMutator(overrideReconcilerAndHydrationResources),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test overriding the cpu request and memory limits of the git-sync container
	overrideGitSyncResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.GitSync,
			CPURequest:    resource.MustParse("200Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideGitSyncResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
		containerResourcesMutator(overrideGitSyncResources),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}

	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestCreateAndUpdateRootReconcilerWithAutopilot(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	overrideReconcilerAndGitSyncResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH),
		rootsyncSecretRef(rootsyncSSHKey), rootsyncOverrideResources(overrideReconcilerAndGitSyncResources))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))
	isAutopilotCluster := true
	testReconciler.isAutopilotCluster = &isAutopilotCluster

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
		containerResourcesMutator(overrideReconcilerAndGitSyncResources),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test Autopilot ignores the resource requirements override.
	overrideReconcilerCPULimitsAndGitSyncMemResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("1.2"),
			CPULimit:      resource.MustParse("1.5"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("0.6"),
			CPULimit:      resource.MustParse("0.7"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			MemoryRequest: resource.MustParse("200Gi"),
			MemoryLimit:   resource.MustParse("222Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerCPULimitsAndGitSyncMemResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully remained unchanged")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}

	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully remained unchanged")
}

func TestUpdateRootReconcilerWithOverrideWithAutopilot(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))
	isAutopilotCluster := true
	testReconciler.isAutopilotCluster = &isAutopilotCluster

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test Autopilot ignores resource requirements override.
	overrideReconcilerAndGitSyncResourceLimits := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			CPURequest:    resource.MustParse("500m"),
			CPULimit:      resource.MustParse("1"),
			MemoryRequest: resource.MustParse("500Mi"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerAndGitSyncResourceLimits,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully remain unchanged")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}

	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully remained unchanged")
}

func TestRootSyncCreateWithNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey), rootsyncNoSSLVerify())
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsUpdatedAnnotationNoSSLVerify)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRootSyncUpdateNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Set rs.Spec.NoSSLVerify to false
	rs.Spec.NoSSLVerify = false
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM := gitSyncConfigMap(ctx, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("No need to update ConfigMap and Deployment")

	// Set rs.Spec.NoSSLVerify to true
	rs.Spec.NoSSLVerify = true
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	updatedRootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsUpdatedAnnotationNoSSLVerify)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Set rs.Spec.NoSSLVerify to false
	rs.Spec.NoSSLVerify = false
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestRootSyncCreateWithOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey), rootsyncOverrideGitSyncDepth(5))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsUpdatedAnnotationOverrideGitSyncDepth)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRootSyncUpdateOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap, ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Test overriding the git sync depth to a positive value
	var depth int64 = 5
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedRootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsUpdatedAnnotationOverrideGitSyncDepth)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment

	updatedCMID, updatedCM := gitSyncConfigMap(ctx, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Test overriding the git sync depth to 0
	depth = 0
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedRootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsUpdatedAnnotationOverrideGitSyncDepthZero)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Set rs.Spec.Override.GitSyncDepth to nil.
	rs.Spec.Override.GitSyncDepth = nil
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("No need to update ConfigMap and Deployment.")
}

func TestRootSyncSwitchAuthTypes(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources with GCPServiceAccount auth type.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs.Name, rootReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	wantServiceAccount := fake.ServiceAccountObject(
		rootReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(kinds.RootSyncV1Beta1().Kind, rootsyncName, ""),
		}),
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotationGCENode)),
		setServiceAccountName(rootReconcilerName),
		gceNodeMutator(rootReconcilerName),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
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

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test updating RootSync resources with None auth type.
	rs.Spec.Auth = noneAuth
	rs.Spec.SecretRef = v1beta1.SecretReference{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotationNone)),
		setServiceAccountName(rootReconcilerName),
		noneMutator(rootReconcilerName),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncReconcilerRestart(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")

	// Scale down the Reconciler Deployment to 0 replicas.
	deploymentCoreObject := fakeClient.Objects[core.IDOf(rootDeployment)]
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
	t.Log("Deployment successfully updated")
}

// This test reconcilers multiple RootSyncs with different auth types.
// - rs1: "my-root-sync", auth type is ssh.
// - rs2: uses the default "root-sync" name and auth type is gcenode
// - rs3: "my-rs-3", auth type is gcpserviceaccount
// - rs4: "my-rs-4", auth type is cookiefile with proxy
// - rs5: "my-rs-5", auth type is token with proxy
func TestMultipleRootSyncs(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs1 := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName1 := namespacedName(rs1.Name, rs1.Namespace)

	rs2 := rootSync(configsync.RootSyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretGCENode))
	reqNamespacedName2 := namespacedName(rs2.Name, rs2.Namespace)

	rs3 := rootSync("my-rs-3", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName3 := namespacedName(rs3.Name, rs3.Namespace)

	rs4 := rootSync("my-rs-4", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretCookieFile), rootsyncSecretRef(reposyncCookie))
	reqNamespacedName4 := namespacedName(rs4.Name, rs4.Namespace)
	secret4 := secretObjWithProxy(t, reposyncCookie, "cookie_file", core.Namespace(rs4.Namespace))

	rs5 := rootSync("my-rs-5", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.GitSecretToken), rootsyncSecretRef(secretName))
	reqNamespacedName5 := namespacedName(rs5.Name, rs5.Namespace)
	secret5 := secretObjWithProxy(t, secretName, GitSecretConfigKeyToken, core.Namespace(rs5.Namespace))
	secret5.Data[GitSecretConfigKeyTokenUsername] = []byte("test-user")

	fakeClient, testReconciler := setupRootReconciler(t, rs1, secretObj(t, rootsyncSSHKey, secretAuth, core.Namespace(rs1.Namespace)))

	rootReconcilerName2 := reconciler.RootReconcilerName(rs2.Name)
	rootReconcilerName3 := reconciler.RootReconcilerName(rs3.Name)
	rootReconcilerName4 := reconciler.RootReconcilerName(rs4.Name)
	rootReconcilerName5 := reconciler.RootReconcilerName(rs5.Name)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.RootReconciler, rs1.Name, rootReconcilerName, &rs1.Spec.Git, &rs1.Spec.Override)

	serviceAccount1 := fake.ServiceAccountObject(
		rootReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(kinds.RootSyncV1Beta1().Kind, rs1.Name, ""),
		}),
	)
	wantServiceAccounts := map[core.ID]*corev1.ServiceAccount{core.IDOf(serviceAccount1): serviceAccount1}

	crb := clusterrolebinding(
		RootSyncPermissionsName(),
		rootReconcilerName,
	)
	rootDeployment1 := rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation(rsAnnotation)),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment1): rootDeployment1}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap, ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Test reconciler rs2: root-sync
	if err := fakeClient.Create(ctx, rs2); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.RootReconciler, rs2.Name, rootReconcilerName2, &rs2.Spec.Git, &rs2.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	rootDeployment2 := rootSyncDeployment(rootReconcilerName2,
		setAnnotations(deploymentAnnotation("6c68a1847f229bcbda53d857b8259e8d")),
		setServiceAccountName(rootReconcilerName2),
		gceNodeMutator(rootReconcilerName2),
	)
	wantDeployments[core.IDOf(rootDeployment2)] = rootDeployment2
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	serviceAccount2 := fake.ServiceAccountObject(
		rootReconcilerName2,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(kinds.RootSyncV1Beta1().Kind, rs2.Name, ""),
		}),
	)
	wantServiceAccounts[core.IDOf(serviceAccount2)] = serviceAccount2
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = append(crb.Subjects, subject(rootReconcilerName2,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

	// Test reconciler rs3: my-rs-3
	if err := fakeClient.Create(ctx, rs3); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.RootReconciler, rs3.Name, rootReconcilerName3, &rs3.Spec.Git, &rs3.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	rootDeployment3 := rootSyncDeployment(rootReconcilerName3,
		setAnnotations(deploymentAnnotation("67cf702aa32e6243d1bf364928322239")),
		setServiceAccountName(rootReconcilerName3),
		gceNodeMutator(rootReconcilerName3),
	)
	wantDeployments[core.IDOf(rootDeployment3)] = rootDeployment3
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Error(err)
	}

	serviceAccount3 := fake.ServiceAccountObject(
		rootReconcilerName3,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(kinds.RootSyncV1Beta1().Kind, rs3.Name, ""),
		}),
		core.Annotation(GCPSAAnnotationKey, rs3.Spec.GCPServiceAccountEmail),
	)
	wantServiceAccounts[core.IDOf(serviceAccount3)] = serviceAccount3
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = append(crb.Subjects, subject(rootReconcilerName3,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

	// Test reconciler rs4: my-rs-4
	if err := fakeClient.Create(ctx, rs4); err != nil {
		t.Fatal(err)
	}
	if err := fakeClient.Create(ctx, secret4); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName4); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.RootReconciler, rs4.Name, rootReconcilerName4, &rs4.Spec.Git, &rs4.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	rootDeployment4 := rootSyncDeployment(rootReconcilerName4,
		setAnnotations(deploymentAnnotation("556bca074173edad69e470be1245e300")),
		setServiceAccountName(rootReconcilerName4),
		secretMutator(rootReconcilerName4, reposyncCookie),
		envVarMutator("HTTPS_PROXY", reposyncCookie, "https_proxy"),
	)
	wantDeployments[core.IDOf(rootDeployment4)] = rootDeployment4
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	serviceAccount4 := fake.ServiceAccountObject(
		rootReconcilerName4,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(kinds.RootSyncV1Beta1().Kind, rs4.Name, ""),
		}),
	)
	wantServiceAccounts[core.IDOf(serviceAccount4)] = serviceAccount4
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = append(crb.Subjects, subject(rootReconcilerName4,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

	// Test reconciler rs5: my-rs-5
	if err := fakeClient.Create(ctx, rs5); err != nil {
		t.Fatal(err)
	}
	if err := fakeClient.Create(ctx, secret5); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName5); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.RootReconciler, rs5.Name, rootReconcilerName5, &rs5.Spec.Git, &rs5.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	rootDeployment5 := rootSyncDeployment(rootReconcilerName5,
		setAnnotations(deploymentAnnotation("9b29cef0765344d4a1f1934fbd115239")),
		setServiceAccountName(rootReconcilerName5),
		secretMutator(rootReconcilerName5, secretName),
		envVarMutator("HTTPS_PROXY", secretName, "https_proxy"),
		envVarMutator(gitSyncName, secretName, GitSecretConfigKeyTokenUsername),
		envVarMutator(gitSyncPassword, secretName, GitSecretConfigKeyToken),
	)
	wantDeployments[core.IDOf(rootDeployment5)] = rootDeployment5
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	serviceAccount5 := fake.ServiceAccountObject(
		rootReconcilerName5,
		core.Namespace(v1.NSConfigManagementSystem),
		core.OwnerReference([]metav1.OwnerReference{
			ownerReference(kinds.RootSyncV1Beta1().Kind, rs5.Name, ""),
		}),
	)
	wantServiceAccounts[core.IDOf(serviceAccount5)] = serviceAccount5
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = append(crb.Subjects, subject(rootReconcilerName5,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

	// Test updating Configmaps and Deployment resources for rs1: my-root-sync
	rs1.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs1); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedGitSyncCMID, updatedGitSyncCM := gitSyncConfigMap(ctx, rootReconcilerName, &rs1.Spec.Git, &rs1.Spec.Override)
	updatedReconcilerCMID, updatedReconcilerCM := reconcilerConfigMap(declared.RootReconciler, rs1.Name, rootReconcilerName, &rs1.Spec.Git)
	wantConfigMaps[updatedGitSyncCMID] = updatedGitSyncCM
	wantConfigMaps[updatedReconcilerCMID] = updatedReconcilerCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	rootDeployment1 = rootSyncDeployment(rootReconcilerName,
		setAnnotations(deploymentAnnotation("76fbcdaf91f8a4de2ac3ce967efce21f")),
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootReconcilerName, rootsyncSSHKey),
	)
	wantDeployments[core.IDOf(rootDeployment1)] = rootDeployment1
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Test updating Configmaps and Deployment resources for rs2: root-sync
	rs2.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs2); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	updatedGitSyncCMID, updatedGitSyncCM = gitSyncConfigMap(ctx, rootReconcilerName2, &rs2.Spec.Git, &rs2.Spec.Override)
	updatedReconcilerCMID, updatedReconcilerCM = reconcilerConfigMap(declared.RootReconciler, rs2.Name, rootReconcilerName2, &rs2.Spec.Git)
	wantConfigMaps[updatedGitSyncCMID] = updatedGitSyncCM
	wantConfigMaps[updatedReconcilerCMID] = updatedReconcilerCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	rootDeployment2 = rootSyncDeployment(rootReconcilerName2,
		setAnnotations(deploymentAnnotation("b52eaf271d623b34d1a167c13e207b3d")),
		setServiceAccountName(rootReconcilerName2),
		gceNodeMutator(rootReconcilerName2),
	)
	wantDeployments[core.IDOf(rootDeployment2)] = rootDeployment2
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Test updating Configmaps and Deployment resources for rs3: my-rs-3
	rs3.Spec.Git.Revision = gitUpdatedRevision
	rs3.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs3); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	updatedGitSyncCMID, updatedGitSyncCM = gitSyncConfigMap(ctx, rootReconcilerName3, &rs3.Spec.Git, &rs3.Spec.Override)
	updatedReconcilerCMID, updatedReconcilerCM = reconcilerConfigMap(declared.RootReconciler, rs3.Name, rootReconcilerName3, &rs3.Spec.Git)
	wantConfigMaps[updatedGitSyncCMID] = updatedGitSyncCM
	wantConfigMaps[updatedReconcilerCMID] = updatedReconcilerCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	rootDeployment3 = rootSyncDeployment(rootReconcilerName3,
		setAnnotations(deploymentAnnotation("ee0db78e0b430944eaf7a23a98f7a111")),
		setServiceAccountName(rootReconcilerName3),
		gceNodeMutator(rootReconcilerName3),
	)
	wantDeployments[core.IDOf(rootDeployment3)] = rootDeployment3
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	// Test garbage collecting ClusterRoleBinding after all RootSyncs are deleted
	if err := fakeClient.Delete(ctx, rs1); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Subject for rs1 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = updateSubjects(crb.Subjects, rs1.Name)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	if err := fakeClient.Delete(ctx, rs2); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Subject for rs2 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = updateSubjects(crb.Subjects, rs2.Name)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	if err := fakeClient.Delete(ctx, rs3); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Subject for rs3 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = updateSubjects(crb.Subjects, rs3.Name)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	if err := fakeClient.Delete(ctx, rs4); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName4); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Subject for rs4 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = updateSubjects(crb.Subjects, rs4.Name)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}

	if err := fakeClient.Delete(ctx, rs5); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName5); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Verify the ClusterRoleBinding of the root-reconciler is deleted
	if err := validateResourceDeleted(crb, fakeClient); err != nil {
		t.Error(err)
	}
}

type depMutator func(*appsv1.Deployment)

func rootSyncDeployment(reconcilerName string, muts ...depMutator) *appsv1.Deployment {
	dep := fake.DeploymentObject(
		core.Namespace(v1.NSConfigManagementSystem),
		core.Name(reconcilerName),
	)
	var replicas int32 = 1
	dep.Spec.Replicas = &replicas
	dep.Annotations = nil
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

func secretMutator(reconcilerName, secretName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = deploymentSecretVolumes(secretName)
		dep.Spec.Template.Spec.Containers = secretMountContainers(reconcilerName)
	}
}

func envVarMutator(envName, secretName, key string) depMutator {
	return func(dep *appsv1.Deployment) {
		for i, con := range dep.Spec.Template.Spec.Containers {
			if con.Name == reconcilermanager.GitSync {
				dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
					Name: envName,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: secretName,
							},
							Key: key,
						},
					},
				})
			}
		}
	}
}

func gceNodeMutator(reconcilerName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "repo"}}
		dep.Spec.Template.Spec.Containers = gceNodeContainers(reconcilerName)
	}
}

func noneMutator(reconcilerName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "repo"}}
		dep.Spec.Template.Spec.Containers = noneContainers(reconcilerName)
	}
}

func setAnnotations(annotations map[string]string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Annotations = annotations
	}
}

func containerResourcesMutator(overrides []v1beta1.ContainerResourcesSpec) depMutator {
	return func(dep *appsv1.Deployment) {
		for _, container := range dep.Spec.Template.Spec.Containers {
			switch container.Name {
			case reconcilermanager.Reconciler, reconcilermanager.GitSync, reconcilermanager.HydrationController:
				for _, override := range overrides {
					if override.ContainerName == container.Name {
						mutateContainerResourceRequestsLimits(&container, override)
					}
				}
			}
		}
	}
}

func mutateContainerResourceRequestsLimits(container *corev1.Container, resourcesSpec v1beta1.ContainerResourcesSpec) {
	if !resourcesSpec.CPURequest.IsZero() {
		container.Resources.Requests[corev1.ResourceCPU] = resourcesSpec.CPURequest
	} else {
		container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse("100m")
	}

	if !resourcesSpec.CPULimit.IsZero() {
		container.Resources.Limits[corev1.ResourceCPU] = resourcesSpec.CPULimit
	} else {
		container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse("100m")
	}

	if !resourcesSpec.MemoryRequest.IsZero() {
		container.Resources.Requests[corev1.ResourceMemory] = resourcesSpec.MemoryRequest
	} else {
		container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse("100Mi")
	}

	if !resourcesSpec.MemoryLimit.IsZero() {
		container.Resources.Limits[corev1.ResourceMemory] = resourcesSpec.MemoryLimit
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
	containers = append(containers, corev1.Container{Name: GceNodeAskpassSidecarName})
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

	if strings.HasPrefix(reconcilerName, reconciler.RootReconcilerPrefix) {
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
