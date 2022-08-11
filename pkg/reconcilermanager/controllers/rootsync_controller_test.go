// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	v1 "kpt.dev/configsync/pkg/api/configmanagement/v1"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/api/configsync/v1beta1"
	hubv1 "kpt.dev/configsync/pkg/api/hub/v1"
	"kpt.dev/configsync/pkg/core"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/reconcilermanager"
	"kpt.dev/configsync/pkg/rootsync"
	syncerFake "kpt.dev/configsync/pkg/syncer/syncertest/fake"
	"kpt.dev/configsync/pkg/testing/fake"
	"kpt.dev/configsync/pkg/validate/raw/validate"
	"sigs.k8s.io/cli-utils/pkg/testutil"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	rootsyncName   = "my-root-sync"
	rootsyncRepo   = "https://github.com/test/rootsync/csp-config-management/"
	rootsyncDir    = "baz-corp"
	testCluster    = "abc-123"
	ociImage       = "gcr.io/stolos-dev/config-sync-ci/kustomize-components"
	helmRepo       = "oci://us-central1-docker.pkg.dev/stolos-dev/helm-oci-1"
	helmChart      = "hello-chart"
	helmVersion    = "0.1.0"
	rootsyncSSHKey = "root-ssh-key"
)

var rootReconcilerName = core.RootReconcilerName(rootsyncName)

func clusterrolebinding(name, reconcilerName string, opts ...core.MetaMutator) *rbacv1.ClusterRoleBinding {
	result := fake.ClusterRoleBindingObject(opts...)
	result.Name = name

	result.RoleRef.Name = "cluster-admin"
	result.RoleRef.Kind = "ClusterRole"
	result.RoleRef.APIGroup = "rbac.authorization.k8s.io"

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

func secretObj(t *testing.T, name string, auth configsync.AuthType, sourceType v1beta1.SourceType, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, "test-key", auth, sourceType)
	return result
}

func secretObjWithProxy(t *testing.T, name string, auth configsync.AuthType, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, "test-key", auth, v1beta1.GitSource)
	m2 := secretData(t, "test-key", "https_proxy", v1beta1.GitSource)
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

func rootsyncSecretType(auth configsync.AuthType) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Auth = auth
	}
}

func rootsyncOCIAuthType(auth configsync.AuthType) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Oci.Auth = auth
	}
}
func rootsyncHelmAuthType(auth configsync.AuthType) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Helm.Auth = auth
	}
}

func rootsyncSecretRef(ref string) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Git.SecretRef = v1beta1.SecretReference{Name: ref}
	}
}

func rootsyncHelmSecretRef(ref string) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Helm.SecretRef = v1beta1.SecretReference{Name: ref}
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

func rootsyncOverrideReconcileTimeout(reconcileTimeout metav1.Duration) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Override.ReconcileTimeout = &reconcileTimeout
	}
}

func rootsyncNoSSLVerify() func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Git.NoSSLVerify = true
	}
}

func rootsyncPrivateCert(privateCertSecret string) func(*v1beta1.RootSync) {
	return func(rs *v1beta1.RootSync) {
		rs.Spec.Git.PrivateCertSecret.Name = privateCertSecret
	}
}

func rootSync(name string, opts ...func(*v1beta1.RootSync)) *v1beta1.RootSync {
	rs := fake.RootSyncObjectV1Beta1(name)
	rs.Spec.SourceType = string(v1beta1.GitSource)
	rs.Spec.Git = &v1beta1.Git{
		Repo: rootsyncRepo,
		Dir:  rootsyncDir,
	}
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func rootSyncWithOCI(name string, opts ...func(*v1beta1.RootSync)) *v1beta1.RootSync {
	rs := fake.RootSyncObjectV1Beta1(name)
	rs.Spec.SourceType = string(v1beta1.OciSource)
	rs.Spec.Oci = &v1beta1.Oci{
		Image: ociImage,
		Dir:   rootsyncDir,
	}
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func rootSyncWithHelm(name string, opts ...func(*v1beta1.RootSync)) *v1beta1.RootSync {
	rs := fake.RootSyncObjectV1Beta1(name)
	rs.Spec.SourceType = string(v1beta1.HelmSource)
	rs.Spec.Helm = &v1beta1.Helm{
		Repo:    helmRepo,
		Chart:   helmChart,
		Version: helmVersion,
	}
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
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerResourcesMutator(overrideAllContainerResources),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully created")

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
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerResourcesMutator(overrideSelectedResources),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
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
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")
}

func TestUpdateRootReconcilerWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully created")

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
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerResourcesMutator(overrideAllContainerResources),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
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
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerResourcesMutator(overrideReconcilerAndHydrationResources),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
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
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerResourcesMutator(overrideGitSyncResources),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
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
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncCreateWithNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey), rootsyncNoSSLVerify())
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully created")
}

func TestRootSyncUpdateNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnv := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully created")

	// Set rs.Spec.NoSSLVerify to false
	rs.Spec.NoSSLVerify = false
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("No need to update Deployment")

	// Set rs.Spec.NoSSLVerify to true
	rs.Spec.NoSSLVerify = true
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnv = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	updatedRootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Set rs.Spec.NoSSLVerify to false
	rs.Spec.NoSSLVerify = false
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncCreateWithPrivateCert(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment
	privateCertSecret := "foo-secret"
	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch),
		rootsyncSecretType(configsync.AuthToken), rootsyncSecretRef(secretName),
		rootsyncPrivateCert(privateCertSecret))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	gitSecret := secretObjWithProxy(t, secretName, GitSecretConfigKeyToken, core.Namespace(rs.Namespace))
	gitSecret.Data[GitSecretConfigKeyTokenUsername] = []byte("test-user")
	fakeClient, testReconciler := setupRootReconciler(t, rs, gitSecret)

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		privateCertSecretMutator(secretName, privateCertSecret),
		envVarMutator("HTTPS_PROXY", secretName, "https_proxy"),
		envVarMutator(gitSyncName, secretName, GitSecretConfigKeyTokenUsername),
		envVarMutator(gitSyncPassword, secretName, GitSecretConfigKeyToken),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")
}

func TestRootSyncUpdatePrivateCert(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.AuthToken), rootsyncSecretRef(secretName))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	gitSecret := secretObjWithProxy(t, secretName, GitSecretConfigKeyToken, core.Namespace(rs.Namespace))
	gitSecret.Data[GitSecretConfigKeyTokenUsername] = []byte("test-user")
	fakeClient, testReconciler := setupRootReconciler(t, rs, gitSecret)

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnv := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(secretName),
		envVarMutator("HTTPS_PROXY", secretName, "https_proxy"),
		envVarMutator(gitSyncName, secretName, GitSecretConfigKeyTokenUsername),
		envVarMutator(gitSyncPassword, secretName, GitSecretConfigKeyToken),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")

	// Unset rs.Spec.PrivateCertSecret
	rs.Spec.PrivateCertSecret.Name = ""
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("No need to update Deployment")

	// Set rs.Spec.PrivateCertSecret
	privateCertSecret := "foo-secret"
	rs.Spec.PrivateCertSecret.Name = privateCertSecret
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnv = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	updatedRootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		privateCertSecretMutator(secretName, privateCertSecret),
		envVarMutator("HTTPS_PROXY", secretName, "https_proxy"),
		envVarMutator(gitSyncName, secretName, GitSecretConfigKeyTokenUsername),
		envVarMutator(gitSyncPassword, secretName, GitSecretConfigKeyToken),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Unset rs.Spec.PrivateCertSecret
	rs.Spec.PrivateCertSecret.Name = ""
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncCreateWithOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey), rootsyncOverrideGitSyncDepth(5))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully created")
}

func TestRootSyncUpdateOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnv := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Test overriding the git sync depth to a positive value
	var depth int64 = 5
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnv = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	updatedRootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Test overriding the git sync depth to 0
	depth = 0
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnv = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	updatedRootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Set rs.Spec.Override.GitSyncDepth to nil.
	rs.Spec.Override.GitSyncDepth = nil
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("No need to update Deployment.")
}

func TestRootSyncCreateWithOverrideReconcileTimeout(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey), rootsyncOverrideReconcileTimeout(metav1.Duration{Duration: 50 * time.Second}))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully created")
}

func TestRootSyncUpdateOverrideReconcileTimeout(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	rootContainerEnv := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Test overriding the reconcile timeout to 50s
	reconcileTimeout := metav1.Duration{Duration: 50 * time.Second}
	rs.Spec.Override.ReconcileTimeout = &reconcileTimeout
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnv = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	updatedRootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv),
	)

	wantDeployments[core.IDOf(updatedRootDeployment)] = updatedRootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Set rs.Spec.Override.ReconcileTimeout to nil.
	rs.Spec.Override.ReconcileTimeout = nil
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("No need to update Deployment.")
}

func TestRootSyncSwitchAuthTypes(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.AuthGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources with GCPServiceAccount auth type.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantServiceAccount := fake.ServiceAccountObject(
		rootReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
		core.Label(metadata.SyncNamespaceLabel, configsync.ControllerNamespace),
		core.Label(metadata.SyncNameLabel, rootsyncName),
	)

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		gceNodeMutator(gcpSAEmail),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	// compare ServiceAccount.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantServiceAccount)], wantServiceAccount, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ServiceAccount diff %s", diff)
	}

	// compare Deployment.
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Resources successfully created")

	// Test updating RootSync resources with SSH auth type.
	rs.Spec.Auth = configsync.AuthSSH
	rs.Spec.Git.SecretRef.Name = rootsyncSSHKey
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnvs = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Test updating RootSync resources with None auth type.
	rs.Spec.Auth = configsync.AuthNone
	rs.Spec.SecretRef = v1beta1.SecretReference{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnvs = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		containersWithRepoVolumeMutator(noneGitContainers()),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncReconcilerRestart(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
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
	if t.Failed() {
		t.FailNow()
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

	rs2 := rootSync(configsync.RootSyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.AuthGCENode))
	reqNamespacedName2 := namespacedName(rs2.Name, rs2.Namespace)

	rs3 := rootSync("my-rs-3", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.AuthGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName3 := namespacedName(rs3.Name, rs3.Namespace)

	rs4 := rootSync("my-rs-4", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.AuthCookieFile), rootsyncSecretRef(reposyncCookie))
	reqNamespacedName4 := namespacedName(rs4.Name, rs4.Namespace)
	secret4 := secretObjWithProxy(t, reposyncCookie, "cookie_file", core.Namespace(rs4.Namespace))

	rs5 := rootSync("my-rs-5", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.AuthToken), rootsyncSecretRef(secretName))
	reqNamespacedName5 := namespacedName(rs5.Name, rs5.Namespace)
	secret5 := secretObjWithProxy(t, secretName, GitSecretConfigKeyToken, core.Namespace(rs5.Namespace))
	secret5.Data[GitSecretConfigKeyTokenUsername] = []byte("test-user")

	fakeClient, testReconciler := setupRootReconciler(t, rs1, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs1.Namespace)))

	rootReconcilerName2 := core.RootReconcilerName(rs2.Name)
	rootReconcilerName3 := core.RootReconcilerName(rs3.Name)
	rootReconcilerName4 := core.RootReconcilerName(rs4.Name)
	rootReconcilerName5 := core.RootReconcilerName(rs5.Name)

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantRootSyncs := map[types.NamespacedName]struct{}{
		{Namespace: rs1.Namespace, Name: rs1.Name}: {},
	}
	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	wantRs1 := fake.RootSyncObjectV1Beta1(rs1.Name)
	wantRs1.Spec = rs1.Spec
	wantRs1.Status.Reconciler = rootReconcilerName
	rootsync.SetReconciling(wantRs1, "Deployment", "Replicas: 0/1")
	controllerutil.AddFinalizer(wantRs1, v1beta1.SyncFinalizer)
	validateRootSyncStatus(t, wantRs1, fakeClient)

	label1 := map[string]string{
		metadata.SyncNamespaceLabel: rs1.Namespace,
		metadata.SyncNameLabel:      rs1.Name,
	}

	serviceAccount1 := fake.ServiceAccountObject(
		rootReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Labels(label1),
	)
	wantServiceAccounts := map[core.ID]*corev1.ServiceAccount{core.IDOf(serviceAccount1): serviceAccount1}

	crb := clusterrolebinding(
		RootSyncPermissionsName(),
		rootReconcilerName,
	)
	crb.Subjects = addSubject(crb.Subjects, rootReconcilerName)
	rootContainerEnv1 := testReconciler.populateContainerEnvs(ctx, rs1, rootReconcilerName)
	rootDeployment1 := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv1),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment1): rootDeployment1}

	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("ServiceAccount, ClusterRoleBinding and Deployment successfully created")

	// Test reconciler rs2: root-sync
	if err := fakeClient.Create(ctx, rs2); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantRootSyncs[types.NamespacedName{Namespace: rs2.Namespace, Name: rs2.Name}] = struct{}{}
	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	wantRs2 := fake.RootSyncObjectV1Beta1(rs2.Name)
	wantRs2.Spec = rs2.Spec
	wantRs2.Status.Reconciler = rootReconcilerName2
	rootsync.SetReconciling(wantRs2, "Deployment", "Replicas: 0/1")
	controllerutil.AddFinalizer(wantRs2, v1beta1.SyncFinalizer)
	validateRootSyncStatus(t, wantRs2, fakeClient)

	label2 := map[string]string{
		metadata.SyncNamespaceLabel: rs2.Namespace,
		metadata.SyncNameLabel:      rs2.Name,
	}

	rootContainerEnv2 := testReconciler.populateContainerEnvs(ctx, rs2, rootReconcilerName2)
	rootDeployment2 := rootSyncDeployment(rootReconcilerName2,
		setServiceAccountName(rootReconcilerName2),
		gceNodeMutator(""),
		containerEnvMutator(rootContainerEnv2),
	)
	wantDeployments[core.IDOf(rootDeployment2)] = rootDeployment2
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	serviceAccount2 := fake.ServiceAccountObject(
		rootReconcilerName2,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Labels(label2),
	)
	wantServiceAccounts[core.IDOf(serviceAccount2)] = serviceAccount2
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = addSubject(crb.Subjects, rootReconcilerName2)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

	// Test reconciler rs3: my-rs-3
	if err := fakeClient.Create(ctx, rs3); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantRootSyncs[types.NamespacedName{Namespace: rs3.Namespace, Name: rs3.Name}] = struct{}{}
	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	wantRs3 := fake.RootSyncObjectV1Beta1(rs3.Name)
	wantRs3.Spec = rs3.Spec
	wantRs3.Status.Reconciler = rootReconcilerName3
	rootsync.SetReconciling(wantRs3, "Deployment", "Replicas: 0/1")
	controllerutil.AddFinalizer(wantRs3, v1beta1.SyncFinalizer)
	validateRootSyncStatus(t, wantRs3, fakeClient)

	label3 := map[string]string{
		metadata.SyncNamespaceLabel: rs3.Namespace,
		metadata.SyncNameLabel:      rs3.Name,
	}

	rootContainerEnv3 := testReconciler.populateContainerEnvs(ctx, rs3, rootReconcilerName3)
	rootDeployment3 := rootSyncDeployment(rootReconcilerName3,
		setServiceAccountName(rootReconcilerName3),
		gceNodeMutator(gcpSAEmail),
		containerEnvMutator(rootContainerEnv3),
	)
	wantDeployments[core.IDOf(rootDeployment3)] = rootDeployment3
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Error(err)
	}

	serviceAccount3 := fake.ServiceAccountObject(
		rootReconcilerName3,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs3.Spec.GCPServiceAccountEmail),
		core.Labels(label3),
	)
	wantServiceAccounts[core.IDOf(serviceAccount3)] = serviceAccount3
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = addSubject(crb.Subjects, rootReconcilerName3)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

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

	wantRootSyncs[types.NamespacedName{Namespace: rs4.Namespace, Name: rs4.Name}] = struct{}{}
	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	wantRs4 := fake.RootSyncObjectV1Beta1(rs4.Name)
	wantRs4.Spec = rs4.Spec
	wantRs4.Status.Reconciler = rootReconcilerName4
	rootsync.SetReconciling(wantRs4, "Deployment", "Replicas: 0/1")
	controllerutil.AddFinalizer(wantRs4, v1beta1.SyncFinalizer)
	validateRootSyncStatus(t, wantRs4, fakeClient)

	label4 := map[string]string{
		metadata.SyncNamespaceLabel: rs4.Namespace,
		metadata.SyncNameLabel:      rs4.Name,
	}

	rootContainerEnvs4 := testReconciler.populateContainerEnvs(ctx, rs4, rootReconcilerName4)
	rootDeployment4 := rootSyncDeployment(rootReconcilerName4,
		setServiceAccountName(rootReconcilerName4),
		secretMutator(reposyncCookie),
		envVarMutator("HTTPS_PROXY", reposyncCookie, "https_proxy"),
		containerEnvMutator(rootContainerEnvs4),
	)
	wantDeployments[core.IDOf(rootDeployment4)] = rootDeployment4
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	serviceAccount4 := fake.ServiceAccountObject(
		rootReconcilerName4,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Labels(label4),
	)
	wantServiceAccounts[core.IDOf(serviceAccount4)] = serviceAccount4
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = addSubject(crb.Subjects, rootReconcilerName4)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

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

	wantRootSyncs[types.NamespacedName{Namespace: rs5.Namespace, Name: rs5.Name}] = struct{}{}
	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	wantRs5 := fake.RootSyncObjectV1Beta1(rs5.Name)
	wantRs5.Spec = rs5.Spec
	wantRs5.Status.Reconciler = rootReconcilerName5
	rootsync.SetReconciling(wantRs5, "Deployment", "Replicas: 0/1")
	controllerutil.AddFinalizer(wantRs5, v1beta1.SyncFinalizer)
	validateRootSyncStatus(t, wantRs5, fakeClient)

	label5 := map[string]string{
		metadata.SyncNamespaceLabel: rs5.Namespace,
		metadata.SyncNameLabel:      rs5.Name,
	}

	rootContainerEnvs5 := testReconciler.populateContainerEnvs(ctx, rs5, rootReconcilerName5)
	rootDeployment5 := rootSyncDeployment(rootReconcilerName5,
		setServiceAccountName(rootReconcilerName5),
		secretMutator(secretName),
		envVarMutator("HTTPS_PROXY", secretName, "https_proxy"),
		envVarMutator(gitSyncName, secretName, GitSecretConfigKeyTokenUsername),
		envVarMutator(gitSyncPassword, secretName, GitSecretConfigKeyToken),
		containerEnvMutator(rootContainerEnvs5),
	)
	wantDeployments[core.IDOf(rootDeployment5)] = rootDeployment5
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	serviceAccount5 := fake.ServiceAccountObject(
		rootReconcilerName5,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Labels(label5),
	)
	wantServiceAccounts[core.IDOf(serviceAccount5)] = serviceAccount5
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	crb.Subjects = addSubject(crb.Subjects, rootReconcilerName5)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

	// Test updating Deployment resources for rs1: my-root-sync
	rs1.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs1); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	rootsync.SetReconciling(wantRs1, "Deployment", "Replicas: 0/1")
	validateRootSyncStatus(t, wantRs1, fakeClient)

	rootContainerEnv1 = testReconciler.populateContainerEnvs(ctx, rs1, rootReconcilerName)
	rootDeployment1 = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnv1),
	)
	wantDeployments[core.IDOf(rootDeployment1)] = rootDeployment1
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Test updating Deployment resources for rs2: root-sync
	rs2.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs2); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	rootsync.SetReconciling(wantRs2, "Deployment", "Replicas: 0/1")
	validateRootSyncStatus(t, wantRs2, fakeClient)

	rootContainerEnv2 = testReconciler.populateContainerEnvs(ctx, rs2, rootReconcilerName2)
	rootDeployment2 = rootSyncDeployment(rootReconcilerName2,
		setServiceAccountName(rootReconcilerName2),
		gceNodeMutator(""),
		containerEnvMutator(rootContainerEnv2),
	)
	wantDeployments[core.IDOf(rootDeployment2)] = rootDeployment2
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Test updating  Deployment resources for rs3: my-rs-3
	rs3.Spec.Git.Revision = gitUpdatedRevision
	rs3.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs3); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	rootsync.SetReconciling(wantRs3, "Deployment", "Replicas: 0/1")
	validateRootSyncStatus(t, wantRs3, fakeClient)

	rootContainerEnv3 = testReconciler.populateContainerEnvs(ctx, rs3, rootReconcilerName3)
	rootDeployment3 = rootSyncDeployment(rootReconcilerName3,
		setServiceAccountName(rootReconcilerName3),
		gceNodeMutator(gcpSAEmail),
		containerEnvMutator(rootContainerEnv3),
	)
	wantDeployments[core.IDOf(rootDeployment3)] = rootDeployment3
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Test garbage collecting ClusterRoleBinding after all RootSyncs are deleted
	if err := fakeClient.Delete(ctx, rs1); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	delete(wantRootSyncs, types.NamespacedName{Namespace: rs1.Namespace, Name: rs1.Name})
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	if err := validateResourceDeleted(core.IDOf(rs1), fakeClient); err != nil {
		t.Error(err)
	}

	// Subject for rs1 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = deleteSubject(crb.Subjects, rootReconcilerName)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	validateRootGeneratedResourcesDeleted(t, fakeClient, rootReconcilerName)
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully delted")

	if err := fakeClient.Delete(ctx, rs2); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	delete(wantRootSyncs, types.NamespacedName{Namespace: rs2.Namespace, Name: rs2.Name})
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	if err := validateResourceDeleted(core.IDOf(rs2), fakeClient); err != nil {
		t.Error(err)
	}

	// Subject for rs2 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = deleteSubject(crb.Subjects, rootReconcilerName2)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	validateRootGeneratedResourcesDeleted(t, fakeClient, rootReconcilerName2)
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully delted")

	if err := fakeClient.Delete(ctx, rs3); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	delete(wantRootSyncs, types.NamespacedName{Namespace: rs3.Namespace, Name: rs3.Name})
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	if err := validateResourceDeleted(core.IDOf(rs3), fakeClient); err != nil {
		t.Error(err)
	}

	// Subject for rs3 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = deleteSubject(crb.Subjects, rootReconcilerName3)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	validateRootGeneratedResourcesDeleted(t, fakeClient, rootReconcilerName3)
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully delted")

	if err := fakeClient.Delete(ctx, rs4); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName4); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	delete(wantRootSyncs, types.NamespacedName{Namespace: rs4.Namespace, Name: rs4.Name})
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	if err := validateResourceDeleted(core.IDOf(rs4), fakeClient); err != nil {
		t.Error(err)
	}

	// Subject for rs4 is removed from ClusterRoleBinding.Subjects
	crb.Subjects = deleteSubject(crb.Subjects, rootReconcilerName4)
	if err := validateClusterRoleBinding(crb, fakeClient); err != nil {
		t.Error(err)
	}
	validateRootGeneratedResourcesDeleted(t, fakeClient, rootReconcilerName4)
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully delted")

	if err := fakeClient.Delete(ctx, rs5); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName5); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare syncs.
	delete(wantRootSyncs, types.NamespacedName{Namespace: rs5.Namespace, Name: rs5.Name})
	if diff := cmp.Diff(testReconciler.syncs, wantRootSyncs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("syncs diff %s", diff)
	}

	if err := validateResourceDeleted(core.IDOf(rs5), fakeClient); err != nil {
		t.Error(err)
	}

	// Verify the ClusterRoleBinding of the root-reconciler is deleted
	if err := validateResourceDeleted(core.IDOf(crb), fakeClient); err != nil {
		t.Error(err)
	}
	validateRootGeneratedResourcesDeleted(t, fakeClient, rootReconcilerName5)
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully delted")
}

func validateRootGeneratedResourcesDeleted(t *testing.T, fakeClient *syncerFake.Client, reconcilerName string) {
	t.Helper()

	// Verify deployment is deleted.
	deployment := fake.DeploymentObject(core.Namespace(configsync.ControllerNamespace), core.Name(reconcilerName))
	if err := validateResourceDeleted(core.IDOf(deployment), fakeClient); err != nil {
		t.Error(err)
	}

	// Verify service account is deleted.
	serviceAccount := fake.ServiceAccountObject(reconcilerName, core.Namespace(configsync.ControllerNamespace))
	if err := validateResourceDeleted(core.IDOf(serviceAccount), fakeClient); err != nil {
		t.Error(err)
	}

	// ReconcilerManager doesn't manage the RootSync Secret
}

func TestMapSecretToRootSyncs(t *testing.T) {
	testSecretName := "ssh-test"
	rootSyncs := map[string][]*v1beta1.RootSync{
		rootsyncSSHKey: {
			rootSync("rs-1", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey)),
			rootSync("rs-2", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(rootsyncSSHKey)),
		},
		testSecretName: {
			rootSync("rs-3", rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(GitSecretConfigKeySSH), rootsyncSecretRef(testSecretName)),
		},
	}
	var expectedRequests = func(secretName string) []reconcile.Request {
		requests := make([]reconcile.Request, len(rootSyncs[secretName]))
		for i, rs := range rootSyncs[secretName] {
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      rs.GetName(),
					Namespace: rs.GetNamespace(),
				},
			}
		}
		return requests
	}

	testCases := []struct {
		name   string
		secret client.Object
		want   []reconcile.Request
	}{
		{
			name:   "A secret from the default namespace",
			secret: fake.SecretObject("s1", core.Namespace("default")),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace starting with %s", configsync.ControllerNamespace, core.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(fmt.Sprintf("%s-bookstore", core.NsReconcilerPrefix), core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A secret 'any' from the %s namespace NOT starting with %s, no mapping RootSync", configsync.ControllerNamespace, core.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject("any", core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A secret %q from the %s namespace NOT starting with %s", rootsyncSSHKey, configsync.ControllerNamespace, core.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(rootsyncSSHKey, core.Namespace(configsync.ControllerNamespace)),
			want:   expectedRequests(rootsyncSSHKey),
		},
		{
			name:   fmt.Sprintf("A secret %q from the %s namespace NOT starting with %s", testSecretName, configsync.ControllerNamespace, core.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(testSecretName, core.Namespace(configsync.ControllerNamespace)),
			want:   expectedRequests(testSecretName),
		},
	}

	var objs []client.Object
	for _, rsList := range rootSyncs {
		for _, rs := range rsList {
			objs = append(objs, rs)
		}
	}
	_, testReconciler := setupRootReconciler(t, objs...)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := testReconciler.mapSecretToRootSyncs(tc.secret)
			if len(tc.want) != len(result) {
				t.Fatalf("%s: expected %d requests, got %d", tc.name, len(tc.want), len(result))
			}
			for _, wantReq := range tc.want {
				found := false
				for _, gotReq := range result {
					if diff := cmp.Diff(wantReq, gotReq); diff == "" {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("%s: expected reques %s doesn't exist in the got requests: %v", tc.name, wantReq, result)
				}
			}
		})
	}
}

func TestInjectFleetWorkloadIdentityCredentialsToRootSync(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSync(rootsyncName, rootsyncRef(gitRevision), rootsyncBranch(branch), rootsyncSecretType(configsync.AuthGCPServiceAccount), rootsyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs, secretObj(t, rootsyncSSHKey, configsync.AuthSSH, v1beta1.GitSource, core.Namespace(rs.Namespace)))
	testReconciler.membership = &hubv1.Membership{
		Spec: hubv1.MembershipSpec{
			Owner: hubv1.MembershipOwner{
				ID: "fakeId",
			},
		},
	}
	// Test creating Deployment resources with GCPServiceAccount auth type.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		gceNodeMutator(gcpSAEmail),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	// compare Deployment.
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Resources successfully created")

	workloadIdentityPool := "test-gke-dev.svc.id.goog"
	testReconciler.membership = &hubv1.Membership{
		Spec: hubv1.MembershipSpec{
			WorkloadIdentityPool: workloadIdentityPool,
			IdentityProvider:     "https://container.googleapis.com/v1/projects/test-gke-dev/locations/us-central1-c/clusters/fleet-workload-identity-test-cluster",
		},
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(map[string]string{
			metadata.FleetWorkloadIdentityCredentials: `{"audience":"identitynamespace:test-gke-dev.svc.id.goog:https://container.googleapis.com/v1/projects/test-gke-dev/locations/us-central1-c/clusters/fleet-workload-identity-test-cluster","credential_source":{"file":"/var/run/secrets/tokens/gcp-ksa/token"},"service_account_impersonation_url":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/config-sync@cs-project.iam.gserviceaccount.com:generateAccessToken","subject_token_type":"urn:ietf:params:oauth:token-type:jwt","token_url":"https://sts.googleapis.com/v1/token","type":"external_account"}`,
		}),
		setServiceAccountName(rootReconcilerName),
		fleetWorkloadIdentityMutator(workloadIdentityPool, gcpSAEmail),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments = map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	// compare Deployment.
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Resources successfully created")

	// Test updating RootSync resources with SSH auth type.
	rs.Spec.Auth = configsync.AuthSSH
	rs.Spec.Git.SecretRef.Name = rootsyncSSHKey
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnvs = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		secretMutator(rootsyncSSHKey),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	// Test updating RootSync resources with None auth type.
	rs.Spec.Auth = configsync.AuthNone
	rs.Spec.SecretRef = v1beta1.SecretReference{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnvs = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		containersWithRepoVolumeMutator(noneGitContainers()),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")
}
func TestRootSyncWithHelm(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = helmParsedDeployment
	secretName := "helm-secret"
	// Test creating RootSync resources with Token auth type
	rs := rootSyncWithHelm(rootsyncName,
		rootsyncHelmAuthType(configsync.AuthToken), rootsyncHelmSecretRef(secretName))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	helmSecret := secretObj(t, secretName, configsync.AuthToken, v1beta1.HelmSource, core.Namespace(rs.Namespace))
	fakeClient, testReconciler := setupRootReconciler(t, rs, helmSecret)

	// Test creating Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		helmSecretMutator(secretName),
		envVarMutator(helmSyncName, secretName, HelmSecretKeyUsername),
		envVarMutator(helmSyncPassword, secretName, HelmSecretKeyPassword),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")

	// Test updating RootSync resources with None auth type.
	rs.Spec.Helm.Auth = configsync.AuthNone
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootContainerEnvs = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		containersWithRepoVolumeMutator(noneHelmContainers()),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}
func TestRootSyncWithOCI(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := rootSyncWithOCI(rootsyncName, rootsyncOCIAuthType(configsync.AuthNone))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs)

	// Test creating Deployment resources with GCPServiceAccount auth type.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantServiceAccount := fake.ServiceAccountObject(
		rootReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Label(metadata.SyncNamespaceLabel, configsync.ControllerNamespace),
		core.Label(metadata.SyncNameLabel, rootsyncName),
	)

	rootContainerEnvs := testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)

	rootDeployment := rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		containersWithRepoVolumeMutator(noneOciContainers()),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(rootDeployment): rootDeployment}

	// compare ServiceAccount.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantServiceAccount)], wantServiceAccount, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ServiceAccount diff %s", diff)
	}

	// compare Deployment.
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Resources successfully created")

	t.Log("Test updating RootSync resources with gcenode auth type.")
	rs.Spec.Oci.Auth = configsync.AuthGCENode
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	// compare ServiceAccount.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantServiceAccount)], wantServiceAccount, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ServiceAccount diff %s", diff)
	}

	rootContainerEnvs = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		containersWithRepoVolumeMutator(noneOciContainers()),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	t.Log("Test updating RootSync resources with gcpserviceaccount auth type.")
	rs.Spec.Oci.Auth = configsync.AuthGCPServiceAccount
	rs.Spec.Oci.GCPServiceAccountEmail = gcpSAEmail
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantServiceAccount = fake.ServiceAccountObject(
		rootReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs.Spec.Oci.GCPServiceAccountEmail),
		core.Label(metadata.SyncNamespaceLabel, configsync.ControllerNamespace),
		core.Label(metadata.SyncNameLabel, rootsyncName),
	)
	// compare ServiceAccount.
	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantServiceAccount)], wantServiceAccount, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ServiceAccount diff %s", diff)
	}
	rootContainerEnvs = testReconciler.populateContainerEnvs(ctx, rs, rootReconcilerName)
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		containersWithRepoVolumeMutator(noneOciContainers()),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	t.Log("Test FWI")
	workloadIdentityPool := "test-gke-dev.svc.id.goog"
	testReconciler.membership = &hubv1.Membership{
		Spec: hubv1.MembershipSpec{
			WorkloadIdentityPool: workloadIdentityPool,
			IdentityProvider:     "https://container.googleapis.com/v1/projects/test-gke-dev/locations/us-central1-c/clusters/fleet-workload-identity-test-cluster",
		},
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if diff := cmp.Diff(fakeClient.Objects[core.IDOf(wantServiceAccount)], wantServiceAccount, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("ServiceAccount diff %s", diff)
	}
	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setAnnotations(map[string]string{
			metadata.FleetWorkloadIdentityCredentials: `{"audience":"identitynamespace:test-gke-dev.svc.id.goog:https://container.googleapis.com/v1/projects/test-gke-dev/locations/us-central1-c/clusters/fleet-workload-identity-test-cluster","credential_source":{"file":"/var/run/secrets/tokens/gcp-ksa/token"},"service_account_impersonation_url":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/config-sync@cs-project.iam.gserviceaccount.com:generateAccessToken","subject_token_type":"urn:ietf:params:oauth:token-type:jwt","token_url":"https://sts.googleapis.com/v1/token","type":"external_account"}`,
		}),
		setServiceAccountName(rootReconcilerName),
		fwiOciMutator(workloadIdentityPool),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")

	t.Log("Test overriding the cpu request and memory limits of the oci-sync container")
	overrideOciSyncResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.OciSync,
			CPURequest:    resource.MustParse("200m"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideOciSyncResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	rootDeployment = rootSyncDeployment(rootReconcilerName,
		setServiceAccountName(rootReconcilerName),
		setAnnotations(map[string]string{
			metadata.FleetWorkloadIdentityCredentials: `{"audience":"identitynamespace:test-gke-dev.svc.id.goog:https://container.googleapis.com/v1/projects/test-gke-dev/locations/us-central1-c/clusters/fleet-workload-identity-test-cluster","credential_source":{"file":"/var/run/secrets/tokens/gcp-ksa/token"},"service_account_impersonation_url":"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/config-sync@cs-project.iam.gserviceaccount.com:generateAccessToken","subject_token_type":"urn:ietf:params:oauth:token-type:jwt","token_url":"https://sts.googleapis.com/v1/token","type":"external_account"}`,
		}),
		fwiOciMutator(workloadIdentityPool),
		containerResourcesMutator(overrideOciSyncResources),
		containerEnvMutator(rootContainerEnvs),
	)
	wantDeployments[core.IDOf(rootDeployment)] = rootDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	if t.Failed() {
		t.FailNow()
	}
	t.Log("Deployment successfully updated")
}

func TestRootSyncSpecValidation(t *testing.T) {
	rs := fake.RootSyncObjectV1Beta1(rootsyncName)
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs)

	// Verify unsupported source type
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs := fake.RootSyncObjectV1Beta1(rootsyncName)
	rootsync.SetStalled(wantRs, "Validation", validate.InvalidSourceType(rs))
	validateRootSyncStatus(t, wantRs, fakeClient)

	// verify missing Git
	rs.Spec.SourceType = string(v1beta1.GitSource)
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs.Spec = rs.Spec
	rootsync.SetStalled(wantRs, "Validation", validate.MissingGitSpec(rs))
	validateRootSyncStatus(t, wantRs, fakeClient)

	// verify missing Oci
	rs.Spec.SourceType = string(v1beta1.OciSource)
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs.Spec = rs.Spec
	rootsync.SetStalled(wantRs, "Validation", validate.MissingOciSpec(rs))
	validateRootSyncStatus(t, wantRs, fakeClient)

	// verify missing OCI image
	rs.Spec.SourceType = string(v1beta1.OciSource)
	rs.Spec.Oci = &v1beta1.Oci{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs.Spec = rs.Spec
	rootsync.SetStalled(wantRs, "Validation", validate.MissingOciImage(rs))
	validateRootSyncStatus(t, wantRs, fakeClient)

	// verify invalid OCI Auth
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment
	rs.Spec.SourceType = string(v1beta1.OciSource)
	rs.Spec.Oci = &v1beta1.Oci{Image: ociImage}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs.Spec = rs.Spec
	rootsync.SetStalled(wantRs, "Validation", validate.InvalidOciAuthType(rs))
	validateRootSyncStatus(t, wantRs, fakeClient)

	// verify redundant source specifications
	rs.Spec.SourceType = string(v1beta1.GitSource)
	rs.Spec.Git = &v1beta1.Git{}
	rs.Spec.Oci = &v1beta1.Oci{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs.Spec = rs.Spec
	rootsync.SetStalled(wantRs, "Validation", validate.RedundantOciSpec(rs))
	validateRootSyncStatus(t, wantRs, fakeClient)

	rs.Spec.SourceType = string(v1beta1.OciSource)
	rs.Spec.Git = &v1beta1.Git{}
	rs.Spec.Oci = &v1beta1.Oci{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs.Spec = rs.Spec
	rootsync.SetStalled(wantRs, "Validation", validate.RedundantGitSpec(rs))
	validateRootSyncStatus(t, wantRs, fakeClient)

	// verify valid OCI spec
	rs.Spec.SourceType = string(v1beta1.OciSource)
	rs.Spec.Git = nil
	rs.Spec.Oci = &v1beta1.Oci{Image: ociImage, Auth: configsync.AuthNone}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	// Clear the stalled condition
	rs.Status = v1beta1.RootSyncStatus{}
	if err := fakeClient.Status().Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the root sync request, got error: %v", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	wantRs.Spec = rs.Spec
	wantRs.Status.Reconciler = rootReconcilerName
	wantRs.Status.Conditions = nil // clear the stalled condition
	rootsync.SetReconciling(wantRs, "Deployment", "Replicas: 0/1")
	controllerutil.AddFinalizer(wantRs, v1beta1.SyncFinalizer)
	validateRootSyncStatus(t, wantRs, fakeClient)
}

func TestRootSyncReconcileStaleClientCache(t *testing.T) {
	rs := fake.RootSyncObjectV1Beta1(rootsyncName)
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupRootReconciler(t, rs)
	ctx := context.Background()

	// Simulate ResourceVersion set by apiserver
	rs.ResourceVersion = "1"
	err := fakeClient.Update(ctx, rs)
	require.NoError(t, err, "unexpected Update error")

	// Reconcile should succeed and update the RootSync
	_, err = testReconciler.Reconcile(ctx, reqNamespacedName)
	require.NoError(t, err, "unexpected Reconcile error")

	// Expect Stalled condition with True status, because the RootSync is invalid
	rs = fake.RootSyncObjectV1Beta1(rootsyncName)
	err = fakeClient.Get(ctx, core.ObjectNamespacedName(rs), rs)
	require.NoError(t, err, "unexpected Get error")
	reconcilingCondition := rootsync.GetCondition(rs.Status.Conditions, v1beta1.RootSyncStalled)
	require.NotNilf(t, reconcilingCondition, "status: %+v", rs.Status)
	require.Equal(t, reconcilingCondition.Status, metav1.ConditionTrue, "unexpected Stalled condition status")
	require.Contains(t, reconcilingCondition.Message, "KNV1061: RootSyncs must specify spec.sourceType", "unexpected Stalled condition message")

	// Expect next Reconcile to error since the ResourceVersion hasn't been updated.
	// This means the client cache hasn't been updated and isn't returning the latest version.
	_, err = testReconciler.Reconcile(ctx, reqNamespacedName)
	require.Error(t, err, "expected Reconcile to error")
	require.Equal(t, err.Error(), "ResourceVersion already reconciled: 1", "unexpected Reconcile error")

	// Simulate ResourceVersion updated in the client cache by the
	// reconciler-manager's resource watch from the apiserver
	rs = fake.RootSyncObjectV1Beta1(rootsyncName)
	err = fakeClient.Get(ctx, core.ObjectNamespacedName(rs), rs)
	require.NoError(t, err, "unexpected Get error")
	rs.ResourceVersion = "A" // doesn't need to be increasing or even numeric
	err = fakeClient.Update(ctx, rs)
	require.NoError(t, err, "unexpected Update error")

	// Reconcile should succeed and NOT update the RootSync
	_, err = testReconciler.Reconcile(ctx, reqNamespacedName)
	require.NoError(t, err, "unexpected Reconcile error")

	// Expect the same Stalled condition error message
	rs = fake.RootSyncObjectV1Beta1(rootsyncName)
	err = fakeClient.Get(ctx, core.ObjectNamespacedName(rs), rs)
	require.NoError(t, err, "unexpected Get error")
	reconcilingCondition = rootsync.GetCondition(rs.Status.Conditions, v1beta1.RootSyncStalled)
	require.NotNilf(t, reconcilingCondition, "status: %+v", rs.Status)
	require.Equal(t, reconcilingCondition.Status, metav1.ConditionTrue, "unexpected Stalled condition status")
	require.Contains(t, reconcilingCondition.Message, "KNV1061: RootSyncs must specify spec.sourceType", "unexpected Stalled condition message")

	// Simulate a spec update, with ResourceVersion updated by the apiserver
	rs = fake.RootSyncObjectV1Beta1(rootsyncName)
	err = fakeClient.Get(ctx, core.ObjectNamespacedName(rs), rs)
	require.NoError(t, err, "unexpected Get error")
	rs.Spec.SourceType = string(v1beta1.GitSource)
	rs.ResourceVersion = "2" // doesn't need to be increasing or even numeric
	err = fakeClient.Update(ctx, rs)
	require.NoError(t, err, "unexpected Update error")

	// Reconcile should succeed and update the RootSync
	_, err = testReconciler.Reconcile(ctx, reqNamespacedName)
	require.NoError(t, err, "unexpected Reconcile error")

	// Expect Stalled condition with True status, because the RootSync is differently invalid
	rs = fake.RootSyncObjectV1Beta1(rootsyncName)
	err = fakeClient.Get(ctx, core.ObjectNamespacedName(rs), rs)
	require.NoError(t, err, "unexpected Get error")
	reconcilingCondition = rootsync.GetCondition(rs.Status.Conditions, v1beta1.RootSyncStalled)
	require.NotNilf(t, reconcilingCondition, "status: %+v", rs.Status)
	require.Equal(t, reconcilingCondition.Status, metav1.ConditionTrue, "unexpected Stalled condition status")
	require.Contains(t, reconcilingCondition.Message, "RootSyncs must specify spec.git when spec.sourceType is \"git\"", "unexpected Stalled condition message")
}

func validateRootSyncStatus(t *testing.T, want *v1beta1.RootSync, fakeClient *syncerFake.Client) {
	gotCoreObject, found := fakeClient.Objects[core.IDOf(want)]
	require.Truef(t, found, "RootSync NotFound: %s/%s", want.Namespace, want.Name)

	got := gotCoreObject.(*v1beta1.RootSync)

	asserter := testutil.NewAsserter(
		cmpopts.IgnoreFields(v1beta1.RootSyncCondition{}, "LastUpdateTime", "LastTransitionTime"))
	// cmpopts.SortSlices(func(x, y v1beta1.RootSyncCondition) bool { return x.Message < y.Message })
	asserter.Equal(t, want.Status.Conditions, got.Status.Conditions, "Unexpected status conditions")
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

func secretMutator(secretName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = deploymentSecretVolumes(secretName, "")
		dep.Spec.Template.Spec.Containers = secretMountContainers("")
	}
}

func helmSecretMutator(secretName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = helmDeploymentSecretVolumes(secretName)
		dep.Spec.Template.Spec.Containers = helmSecretMountContainers()
	}
}

func privateCertSecretMutator(secretName, privateCertSecretName string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = deploymentSecretVolumes(secretName, privateCertSecretName)
		dep.Spec.Template.Spec.Containers = secretMountContainers(privateCertSecretName)
	}
}

func envVarMutator(envName, secretName, key string) depMutator {
	return func(dep *appsv1.Deployment) {
		for i, con := range dep.Spec.Template.Spec.Containers {
			if con.Name == reconcilermanager.GitSync || con.Name == reconcilermanager.HelmSync {
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

func containerEnvMutator(containerEnvs map[string][]corev1.EnvVar) depMutator {
	return func(dep *appsv1.Deployment) {
		for i, con := range dep.Spec.Template.Spec.Containers {
			dep.Spec.Template.Spec.Containers[i].Env = append(dep.Spec.Template.Spec.Containers[i].Env, containerEnvs[con.Name]...)
		}
	}
}

func gceNodeMutator(gsaEmail string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "repo"}}
		dep.Spec.Template.Spec.Containers = gceNodeContainers(gsaEmail)
	}
}

func fwiVolume(workloadIdentityPool string) corev1.Volume {
	return corev1.Volume{
		Name: gcpKSAVolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Audience:          workloadIdentityPool,
							ExpirationSeconds: &expirationSeconds,
							Path:              gsaTokenPath,
						},
					},
					{
						DownwardAPI: &corev1.DownwardAPIProjection{Items: []corev1.DownwardAPIVolumeFile{
							{
								Path: googleApplicationCredentialsFile,
								FieldRef: &corev1.ObjectFieldSelector{
									APIVersion: "v1",
									FieldPath:  fmt.Sprintf("metadata.annotations['%s']", metadata.FleetWorkloadIdentityCredentials),
								},
							},
						}},
					},
				},
				DefaultMode: &defaultMode,
			},
		},
	}
}

func fleetWorkloadIdentityMutator(workloadIdentityPool, gsaEmail string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: "repo"},
			fwiVolume(workloadIdentityPool),
		}
		dep.Spec.Template.Spec.Containers = fleetWorkloadIdentityContainers(gsaEmail)
	}
}

func fleetWorkloadIdentityContainers(gsaEmail string) []corev1.Container {
	containers := noneGitContainers()
	containers = append(containers, corev1.Container{
		Name: GceNodeAskpassSidecarName,
		Env: []corev1.EnvVar{{
			Name:  googleApplicationCredentialsEnvKey,
			Value: filepath.Join(gcpKSATokenDir, googleApplicationCredentialsFile),
		}, {
			Name:  gsaEmailEnvKey,
			Value: gsaEmail,
		}},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      gcpKSAVolumeName,
			ReadOnly:  true,
			MountPath: gcpKSATokenDir,
		}},
	})
	return containers
}

func fwiOciMutator(workloadIdentityPool string) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{
			{Name: "repo"},
			fwiVolume(workloadIdentityPool),
		}
		dep.Spec.Template.Spec.Containers = fwiOciContainers()
	}
}

func fwiOciContainers() []corev1.Container {
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
			Name:      reconcilermanager.OciSync,
			Resources: defaultResourceRequirements(),
			Env: []corev1.EnvVar{{
				Name:  googleApplicationCredentialsEnvKey,
				Value: filepath.Join(gcpKSATokenDir, googleApplicationCredentialsFile),
			}},
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
				{
					Name:      gcpKSAVolumeName,
					ReadOnly:  true,
					MountPath: gcpKSATokenDir,
				}},
		},
	}
}

func containersWithRepoVolumeMutator(containers []corev1.Container) depMutator {
	return func(dep *appsv1.Deployment) {
		dep.Spec.Template.Spec.Volumes = []corev1.Volume{{Name: "repo"}}
		dep.Spec.Template.Spec.Containers = containers
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
			case reconcilermanager.Reconciler, reconcilermanager.GitSync, reconcilermanager.HydrationController, reconcilermanager.OciSync:
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
			},
		},
		{
			Name:      reconcilermanager.OciSync,
			Resources: defaultResourceRequirements(),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
			},
		},
		{
			Name:      reconcilermanager.HelmSync,
			Resources: defaultResourceRequirements(),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
				{Name: "helm-creds", MountPath: "/etc/helm-secret", ReadOnly: true},
			},
		},
	}
}

func secretMountContainers(privateCertSecret string) []corev1.Container {
	gitSyncVolumeMounts := []corev1.VolumeMount{
		{Name: "repo", MountPath: "/repo"},
		{Name: "git-creds", MountPath: "/etc/git-secret", ReadOnly: true},
	}
	if privateCertSecret != "" {
		gitSyncVolumeMounts = append(gitSyncVolumeMounts, corev1.VolumeMount{
			Name: "private-cert", MountPath: "/etc/private-cert", ReadOnly: true,
		})
	}
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
			Name:         reconcilermanager.GitSync,
			Resources:    defaultResourceRequirements(),
			VolumeMounts: gitSyncVolumeMounts,
		},
	}
}

func helmSecretMountContainers() []corev1.Container {
	helmSyncVolumeMounts := []corev1.VolumeMount{
		{Name: "repo", MountPath: "/repo"},
		{Name: "helm-creds", MountPath: "/etc/helm-secret", ReadOnly: true},
	}
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
			Name:         reconcilermanager.HelmSync,
			Resources:    defaultResourceRequirements(),
			VolumeMounts: helmSyncVolumeMounts,
		},
	}
}

func noneGitContainers() []corev1.Container {
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
			}},
	}
}

func noneOciContainers() []corev1.Container {
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
			Name:      reconcilermanager.OciSync,
			Resources: defaultResourceRequirements(),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
			}},
	}
}

func noneHelmContainers() []corev1.Container {
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
			Name:      reconcilermanager.HelmSync,
			Resources: defaultResourceRequirements(),
			VolumeMounts: []corev1.VolumeMount{
				{Name: "repo", MountPath: "/repo"},
			}},
	}
}

func gceNodeContainers(gsaEmail string) []corev1.Container {
	containers := noneGitContainers()
	containers = append(containers, corev1.Container{
		Name: GceNodeAskpassSidecarName,
		Env:  []corev1.EnvVar{{Name: gsaEmailEnvKey, Value: gsaEmail}},
	})
	return containers
}

func deploymentSecretVolumes(secretName, privateCertSecretName string) []corev1.Volume {
	volumes := []corev1.Volume{
		{Name: "repo"},
		{Name: "git-creds", VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		}},
	}
	if usePrivateCert(privateCertSecretName) {
		volumes = append(volumes, corev1.Volume{
			Name: "private-cert", VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: privateCertSecretName,
					Items: []corev1.KeyToPath{
						{
							Key:  PrivateCertKey,
							Path: PrivateCertKey,
						},
					},
					DefaultMode: &defaultMode},
			},
		})
	}
	return volumes
}

func helmDeploymentSecretVolumes(secretName string) []corev1.Volume {
	volumes := []corev1.Volume{
		{Name: "repo"},
		{Name: "helm-creds", VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		}},
	}
	return volumes
}
