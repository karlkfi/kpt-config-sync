package controllers

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	auth               = "ssh"
	branch             = "1.0.0"
	gitRevision        = "1.0.0.rc.8"
	gitUpdatedRevision = "1.1.0.rc.1"

	reposyncNs     = "bookinfo"
	reposyncName   = "my-repo-sync"
	reposyncRepo   = "https://github.com/test/reposync/csp-config-management/"
	reposyncDir    = "foo-corp"
	reposyncSSHKey = "ssh-key"
	reposyncCookie = "cookie"

	secretName = "git-creds"

	gcpSAEmail = "config-sync@cs-project.iam.gserviceaccount.com"

	pollingPeriod = "50ms"

	// Hash of all configmap.data created by Namespace Reconciler.
	nsAnnotation = "f78728fc222ddf41a755df96c1a6708c"
	// Updated hash of all configmap.data updated by Namespace Reconciler.
	nsUpdatedAnnotation = "b956816112b01e4f09139a8aa0c7a158"

	nsUpdatedAnnotationOverrideGitSyncDepth     = "57e9e805fc7b8433368a8007d6c66cfc"
	nsUpdatedAnnotationOverrideGitSyncDepthZero = "1b3e57004ca65924c99136e349963997"

	nsUpdatedAnnotationNoSSLVerify = "c4bb62c2a484b6aca064493efefe9263"

	nsAnnotationGCENode = "fd8889c6bdfa64e4c45249da9d0a1cc9"
	nsAnnotationNone    = "bfc6701fad5bcb7b04a7a16f0bd1ddcf"
)

// Set in init.
var filesystemPollingPeriod time.Duration
var hydrationPollingPeriod time.Duration
var nsReconcilerName = reconciler.NsReconcilerName(reposyncNs, reposyncName)

var parsedDeployment = func(de *appsv1.Deployment) error {
	de.TypeMeta = fake.ToTypeMeta(kinds.Deployment())
	de.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				metadata.ReconcilerLabel: reconcilermanager.Reconciler,
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
		klog.Exitf("failed to parse polling period: %q, got error: %v, want error: nil", pollingPeriod, err)
	}
	hydrationPollingPeriod = filesystemPollingPeriod
}

func reposyncRef(rev string) func(*v1beta1.RepoSync) {
	return func(rs *v1beta1.RepoSync) {
		rs.Spec.Revision = rev
	}
}

func reposyncBranch(branch string) func(*v1beta1.RepoSync) {
	return func(rs *v1beta1.RepoSync) {
		rs.Spec.Branch = branch
	}
}

func reposyncSecretType(auth string) func(*v1beta1.RepoSync) {
	return func(rs *v1beta1.RepoSync) {
		rs.Spec.Auth = auth
	}
}

func reposyncSecretRef(ref string) func(*v1beta1.RepoSync) {
	return func(rs *v1beta1.RepoSync) {
		rs.Spec.Git.SecretRef = v1beta1.SecretReference{Name: ref}
	}
}

func reposyncGCPSAEmail(email string) func(sync *v1beta1.RepoSync) {
	return func(sync *v1beta1.RepoSync) {
		sync.Spec.GCPServiceAccountEmail = email
	}
}

func reposyncOverrideResources(containers []v1beta1.ContainerResourcesSpec) func(sync *v1beta1.RepoSync) {
	return func(sync *v1beta1.RepoSync) {
		sync.Spec.Override = v1beta1.OverrideSpec{
			Resources: containers,
		}
	}
}

func reposyncOverrideGitSyncDepth(depth int64) func(*v1beta1.RepoSync) {
	return func(rs *v1beta1.RepoSync) {
		rs.Spec.Override.GitSyncDepth = &depth
	}
}

func reposyncNoSSLVerify() func(*v1beta1.RepoSync) {
	return func(rs *v1beta1.RepoSync) {
		rs.Spec.NoSSLVerify = true
	}
}

func repoSync(ns, name string, opts ...func(*v1beta1.RepoSync)) *v1beta1.RepoSync {
	rs := fake.RepoSyncObjectV1Beta1(ns, name)
	rs.Spec.Repo = reposyncRepo
	rs.Spec.Dir = reposyncDir
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func rolebinding(name, reconcilerName string, opts ...core.MetaMutator) *rbacv1.RoleBinding {
	result := fake.RoleBindingObject(opts...)
	result.Name = name

	result.RoleRef.Name = RepoSyncPermissionsName()
	result.RoleRef.Kind = "ClusterRole"
	result.RoleRef.APIGroup = "rbac.authorization.k8s.io"

	var sub rbacv1.Subject
	sub.Kind = "ServiceAccount"
	sub.Name = reconcilerName
	sub.Namespace = configsync.ControllerNamespace
	result.Subjects = append(result.Subjects, sub)

	return result
}

func deploymentAnnotation(value string) map[string]string {
	return map[string]string{
		metadata.ConfigMapAnnotationKey: value,
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
	if err := admissionregistrationv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}

	fakeClient := syncerFake.NewClient(t, s, objs...)
	testReconciler := NewRepoSyncReconciler(
		testCluster,
		filesystemPollingPeriod,
		hydrationPollingPeriod,
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RepoSync"),
		s,
	)
	return fakeClient, testReconciler
}

func TestCreateAndUpdateNamespaceReconcilerWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

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

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth),
		reposyncSecretRef(reposyncSSHKey), reposyncOverrideResources(overrideReconcilerAndGitSyncResourceLimits))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		containerResourcesMutator(overrideReconcilerAndGitSyncResourceLimits),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test overriding the CPU resources of the reconciler container and the memory resources of the git-sync container
	overrideReconcilerCPUAndGitSyncMemResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("0.8"),
			CPULimit:      resource.MustParse("1.2"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("0.6"),
			CPULimit:      resource.MustParse("0.8"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			MemoryRequest: resource.MustParse("777Gi"),
			MemoryLimit:   resource.MustParse("888Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerCPUAndGitSyncMemResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		containerResourcesMutator(overrideReconcilerCPUAndGitSyncMemResources),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestUpdateNamespaceReconcilerWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test overriding the CPU/memory limits of both the reconciler and git-sync container
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

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerAndGitSyncResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		containerResourcesMutator(overrideReconcilerAndGitSyncResources),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test overriding the CPU/memory requests and limits of the reconciler container
	overrideReconcilerResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.Reconciler,
			CPURequest:    resource.MustParse("1.8"),
			CPULimit:      resource.MustParse("2"),
			MemoryRequest: resource.MustParse("1.8Gi"),
			MemoryLimit:   resource.MustParse("2Gi"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("1"),
			CPULimit:      resource.MustParse("1.3"),
			MemoryRequest: resource.MustParse("3Gi"),
			MemoryLimit:   resource.MustParse("4Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		containerResourcesMutator(overrideReconcilerResources),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test overriding the memory requests and limits of the git-sync container
	overrideGitSyncResources := []v1beta1.ContainerResourcesSpec{
		{
			ContainerName: reconcilermanager.GitSync,
			MemoryRequest: resource.MustParse("800m"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideGitSyncResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		containerResourcesMutator(overrideGitSyncResources),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestCreateAndUpdateNamespaceReconcilerWithAutopilot(t *testing.T) {
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

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth),
		reposyncSecretRef(reposyncSSHKey), reposyncOverrideResources(overrideReconcilerAndGitSyncResources))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))
	isAutopilotCluster := true
	testReconciler.isAutopilotCluster = &isAutopilotCluster

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		containerResourcesMutator(overrideReconcilerAndGitSyncResources),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

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
			CPURequest:    resource.MustParse("1"),
			CPULimit:      resource.MustParse("1.2"),
		},
		{
			ContainerName: reconcilermanager.HydrationController,
			CPURequest:    resource.MustParse("0.6"),
			CPULimit:      resource.MustParse("0.8"),
		},
		{
			ContainerName: reconcilermanager.GitSync,
			MemoryRequest: resource.MustParse("666Gi"),
			MemoryLimit:   resource.MustParse("888Gi"),
		},
	}

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerCPULimitsAndGitSyncMemResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
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
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully remained unchanged")
}

func TestUpdateNamespaceReconcilerWithOverrideWithAutopilot(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))
	isAutopilotCluster := true
	testReconciler.isAutopilotCluster = &isAutopilotCluster

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test Autopilot ignores resource requirements override.
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

	rs.Spec.Override = v1beta1.OverrideSpec{
		Resources: overrideReconcilerAndGitSyncResources,
	}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully remained unchanged")
}

func TestRepoSyncCreateWithNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey), reposyncNoSSLVerify())
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsUpdatedAnnotationNoSSLVerify)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRepoSyncUpdateNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM := gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM

	updatedRepoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsUpdatedAnnotationNoSSLVerify)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments[core.IDOf(updatedRepoDeployment)] = updatedRepoDeployment

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Set rs.Spec.NoSSLVerify to false
	rs.Spec.NoSSLVerify = false
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestRepoSyncCreateWithOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey), reposyncOverrideGitSyncDepth(5))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsUpdatedAnnotationOverrideGitSyncDepth)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRepoSyncUpdateOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")

	// Test overriding the git sync depth to a positive value
	var depth int64 = 5
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM := gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM

	updatedRepoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsUpdatedAnnotationOverrideGitSyncDepth)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments[core.IDOf(repoDeployment)] = updatedRepoDeployment

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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM

	updatedRepoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsUpdatedAnnotationOverrideGitSyncDepthZero)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments[core.IDOf(repoDeployment)] = updatedRepoDeployment

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
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1beta1.OverrideSpec{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v, want error: nil", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("No need to update ConfigMap and Deployment.")
}

func TestRepoSyncSwitchAuthTypes(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretGCPServiceAccount), reposyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources with GCPServiceAccount auth type.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantNamespaces := map[string]struct{}{
		rs.Namespace: {},
	}

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs.Namespace), rs.Name, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)

	wantServiceAccount := fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotationGCENode)),
		setServiceAccountName(nsReconcilerName),
		gceNodeMutator(nsReconcilerName),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

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

	// Test updating RepoSync resources with SSH auth type.
	rs.Spec.Auth = secretAuth
	rs.Spec.Git.SecretRef.Name = reposyncSSHKey
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment

	updatedCMID, updatedCM := gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test updating RepoSync resources with None auth type.
	rs.Spec.Auth = noneAuth
	rs.Spec.SecretRef = v1beta1.SecretReference{}
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotationNone)),
		setServiceAccountName(nsReconcilerName),
		noneMutator(nsReconcilerName),
	)
	wantDeployments[core.IDOf(repoDeployment)] = repoDeployment

	updatedCMID, updatedCM = gitSyncConfigMap(ctx, nsReconcilerName, &rs.Spec.Git, &rs.Spec.Override)
	wantConfigMaps[updatedCMID] = updatedCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")
}

func TestRepoSyncReconcilerRestart(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(rs.Name, rs.Namespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs.Namespace)))

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	repoDeployment := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment): repoDeployment}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully created")

	// Scale down the Reconciler Deployment to 0 replicas.
	deploymentCoreObject := fakeClient.Objects[core.IDOf(repoDeployment)]
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
	t.Log("Deployment successfully updated")
}

// This test reconcilers multiple RepoSyncs with different auth types.
// - rs1: "my-repo-sync", namespace is bookinfo, auth type is ssh.
// - rs2: uses the default "repo-sync" name, namespace is videoinfo, and auth type is gcenode
// - rs3: "my-rs-3", namespace is videoinfo, auth type is gcpserviceaccount
// - rs4: "my-rs-4", namespace is bookinfo, auth type is cookiefile with proxy
// - rs5: "my-rs-5", namespace is bookinfo, auth type is token with proxy
func TestMultipleRepoSyncs(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	ns2 := "videoinfo"
	rs1 := repoSync(reposyncNs, reposyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName1 := namespacedName(rs1.Name, rs1.Namespace)

	rs2 := repoSync(ns2, configsync.RepoSyncName, reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretGCENode))
	reqNamespacedName2 := namespacedName(rs2.Name, rs2.Namespace)

	rs3 := repoSync(ns2, "my-rs-3", reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretGCPServiceAccount), reposyncGCPSAEmail(gcpSAEmail))
	reqNamespacedName3 := namespacedName(rs3.Name, rs3.Namespace)

	rs4 := repoSync(reposyncNs, "my-rs-4", reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretCookieFile), reposyncSecretRef(reposyncCookie))
	secret4 := secretObjWithProxy(t, reposyncCookie, "cookie_file", core.Namespace(rs4.Namespace))
	reqNamespacedName4 := namespacedName(rs4.Name, rs4.Namespace)

	rs5 := repoSync(reposyncNs, "my-rs-5", reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretToken), reposyncSecretRef(secretName))
	reqNamespacedName5 := namespacedName(rs5.Name, rs5.Namespace)
	secret5 := secretObjWithProxy(t, secretName, GitSecretConfigKeyToken, core.Namespace(rs5.Namespace))
	secret5.Data[GitSecretConfigKeyTokenUsername] = []byte("test-user")

	fakeClient, testReconciler := setupNSReconciler(t, rs1, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(rs1.Namespace)))

	nsReconcilerName2 := reconciler.NsReconcilerName(rs2.Namespace, rs2.Name)
	nsReconcilerName3 := reconciler.NsReconcilerName(rs3.Namespace, rs3.Name)
	nsReconcilerName4 := reconciler.NsReconcilerName(rs4.Namespace, rs4.Name)
	nsReconcilerName5 := reconciler.NsReconcilerName(rs5.Namespace, rs5.Name)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantNamespaces := map[string]struct{}{
		rs1.Namespace: {},
	}

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	wantConfigMaps := buildWantConfigMaps(ctx, declared.Scope(rs1.Namespace), rs1.Name, nsReconcilerName, &rs1.Spec.Git, &rs1.Spec.Override)

	serviceAccount1 := fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
	)
	wantServiceAccounts := map[core.ID]*corev1.ServiceAccount{core.IDOf(serviceAccount1): serviceAccount1}

	roleBinding1 := rolebinding(
		RepoSyncPermissionsName(),
		nsReconcilerName,
		core.Namespace(rs1.Namespace),
	)
	wantRoleBindings := map[core.ID]*rbacv1.RoleBinding{core.IDOf(roleBinding1): roleBinding1}

	repoDeployment1 := repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments := map[core.ID]*appsv1.Deployment{core.IDOf(repoDeployment1): repoDeployment1}

	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap, ServiceAccount, RoleBinding, Deployment successfully created")

	// Test reconciler rs2: repo-sync
	if err := fakeClient.Create(ctx, rs2); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantNamespaces[rs2.Namespace] = struct{}{}
	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}
	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.Scope(rs2.Namespace), rs2.Name, nsReconcilerName2, &rs2.Spec.Git, &rs2.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	repoDeployment2 := repoSyncDeployment(
		nsReconcilerName2,
		setAnnotations(deploymentAnnotation("e873eab42a987877b111e1bfe62a7d48")),
		setServiceAccountName(nsReconcilerName2),
		gceNodeMutator(nsReconcilerName2),
	)
	wantDeployments[core.IDOf(repoDeployment2)] = repoDeployment2
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	serviceAccount2 := fake.ServiceAccountObject(
		nsReconcilerName2,
		core.Namespace(v1.NSConfigManagementSystem),
	)
	wantServiceAccounts[core.IDOf(serviceAccount2)] = serviceAccount2
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	roleBinding2 := rolebinding(
		RepoSyncPermissionsName(),
		nsReconcilerName2,
		core.Namespace(rs2.Namespace),
	)
	wantRoleBindings[core.IDOf(roleBinding2)] = roleBinding2
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and RoleBindings successfully created")

	// Test reconciler rs3: my-rs-3
	if err := fakeClient.Create(ctx, rs3); err != nil {
		t.Fatal(err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}
	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.Scope(rs3.Namespace), rs3.Name, nsReconcilerName3, &rs3.Spec.Git, &rs3.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	repoDeployment3 := repoSyncDeployment(
		nsReconcilerName3,
		setAnnotations(deploymentAnnotation("d0f08f589e7c59d169165f5c7721e7a1")),
		setServiceAccountName(nsReconcilerName3),
		gceNodeMutator(nsReconcilerName3),
	)
	wantDeployments[core.IDOf(repoDeployment3)] = repoDeployment3
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	serviceAccount3 := fake.ServiceAccountObject(
		nsReconcilerName3,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs3.Spec.GCPServiceAccountEmail),
	)
	wantServiceAccounts[core.IDOf(serviceAccount3)] = serviceAccount3
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	// Add to roleBinding2.Subjects because rs3 and rs2 are in the same namespace.
	roleBinding2.Subjects = append(roleBinding2.Subjects, subject(nsReconcilerName3,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and RoleBindings successfully created")

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

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.Scope(rs4.Namespace), rs4.Name, nsReconcilerName4, &rs4.Spec.Git, &rs4.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	repoDeployment4 := repoSyncDeployment(
		nsReconcilerName4,
		setAnnotations(deploymentAnnotation("269ef1cad67dc3e9d3ce2644bb714e5d")),
		setServiceAccountName(nsReconcilerName4),
		secretMutator(nsReconcilerName4, nsReconcilerName4+"-"+reposyncCookie),
		envVarMutator("HTTPS_PROXY", nsReconcilerName4+"-"+reposyncCookie, "https_proxy"),
	)
	wantDeployments[core.IDOf(repoDeployment4)] = repoDeployment4
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}

	serviceAccount4 := fake.ServiceAccountObject(
		nsReconcilerName4,
		core.Namespace(v1.NSConfigManagementSystem),
	)
	wantServiceAccounts[core.IDOf(serviceAccount4)] = serviceAccount4
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	// Add to roleBinding1.Subjects because rs1 and rs4 are in the same namespace.
	roleBinding1.Subjects = append(roleBinding1.Subjects, subject(nsReconcilerName4,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and RoleBindings successfully created")

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

	// compare namespaces.
	if diff := cmp.Diff(testReconciler.namespaces, wantNamespaces, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("namespaces diff %s", diff)
	}

	addConfigMaps(wantConfigMaps, buildWantConfigMaps(ctx, declared.Scope(rs5.Namespace), rs5.Name, nsReconcilerName5, &rs5.Spec.Git, &rs5.Spec.Override))
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	repoDeployment5 := repoSyncDeployment(
		nsReconcilerName5,
		setAnnotations(deploymentAnnotation("f5038ac462d1ea5f4af8b4c7f0b1e64d")),
		setServiceAccountName(nsReconcilerName5),
		secretMutator(nsReconcilerName5, nsReconcilerName5+"-"+secretName),
		envVarMutator("HTTPS_PROXY", nsReconcilerName5+"-"+secretName, "https_proxy"),
		envVarMutator(gitSyncName, nsReconcilerName5+"-"+secretName, GitSecretConfigKeyTokenUsername),
		envVarMutator(gitSyncPassword, nsReconcilerName5+"-"+secretName, GitSecretConfigKeyToken),
	)
	wantDeployments[core.IDOf(repoDeployment5)] = repoDeployment5
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	serviceAccount5 := fake.ServiceAccountObject(
		nsReconcilerName5,
		core.Namespace(v1.NSConfigManagementSystem),
	)
	wantServiceAccounts[core.IDOf(serviceAccount5)] = serviceAccount5
	if err := validateServiceAccounts(wantServiceAccounts, fakeClient); err != nil {
		t.Error(err)
	}

	// Add to roleBinding1.Subjects because rs1 and rs5 are in the same namespace.
	roleBinding1.Subjects = append(roleBinding1.Subjects, subject(nsReconcilerName5,
		configsync.ControllerNamespace,
		"ServiceAccount"))
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}

	t.Log("ConfigMaps, Deployments, ServiceAccounts, and ClusterRoleBindings successfully created")

	// Test updating Configmaps and Deployment resources for rs1: my-repo-sync
	rs1.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs1); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	repoDeployment1 = repoSyncDeployment(
		nsReconcilerName,
		setAnnotations(deploymentAnnotation(nsUpdatedAnnotation)),
		setServiceAccountName(nsReconcilerName),
		secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
	)
	wantDeployments[core.IDOf(repoDeployment1)] = repoDeployment1

	updatedGitSyncCMID, updatedGitSyncCM := gitSyncConfigMap(ctx, nsReconcilerName, &rs1.Spec.Git, &rs1.Spec.Override)
	updatedReconcilerCMID, updatedReconcilerCM := reconcilerConfigMap(declared.Scope(rs1.Namespace), rs1.Name, nsReconcilerName, &rs1.Spec.Git)
	wantConfigMaps[updatedGitSyncCMID] = updatedGitSyncCM
	wantConfigMaps[updatedReconcilerCMID] = updatedReconcilerCM
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")

	// Test updating Configmaps and Deployment resources for rs2: repo-sync
	rs2.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs2); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedGitSyncCMID2, updatedGitSyncCM2 := gitSyncConfigMap(ctx, nsReconcilerName2, &rs2.Spec.Git, &rs2.Spec.Override)
	updatedReconcilerCMID2, updatedReconcilerCM2 := reconcilerConfigMap(declared.Scope(rs2.Namespace), rs2.Name, nsReconcilerName2, &rs2.Spec.Git)
	wantConfigMaps[updatedGitSyncCMID2] = updatedGitSyncCM2
	wantConfigMaps[updatedReconcilerCMID2] = updatedReconcilerCM2
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	repoDeployment2 = repoSyncDeployment(
		nsReconcilerName2,
		setAnnotations(deploymentAnnotation("e29ef13dbdbdf1d6f9b9aec22e6176c0")),
		setServiceAccountName(nsReconcilerName2),
		gceNodeMutator(nsReconcilerName2),
	)
	wantDeployments[core.IDOf(repoDeployment2)] = repoDeployment2

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully updated")

	// Test updating Configmaps and Deployment resources for rs3: my-rs-3
	rs3.Spec.Git.Revision = gitUpdatedRevision
	if err := fakeClient.Update(ctx, rs3); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	updatedGitSyncCMID3, updatedGitSyncCM3 := gitSyncConfigMap(ctx, nsReconcilerName3, &rs3.Spec.Git, &rs3.Spec.Override)
	updatedReconcilerCMID3, updatedReconcilerCM3 := reconcilerConfigMap(declared.Scope(rs3.Namespace), rs3.Name, nsReconcilerName3, &rs3.Spec.Git)
	wantConfigMaps[updatedGitSyncCMID3] = updatedGitSyncCM3
	wantConfigMaps[updatedReconcilerCMID3] = updatedReconcilerCM3
	if err := validateConfigMaps(wantConfigMaps, fakeClient); err != nil {
		t.Error(err)
	}

	repoDeployment3 = repoSyncDeployment(
		nsReconcilerName3,
		setAnnotations(deploymentAnnotation("a984da7a9e0ed7002c8d2ce24b87a485")),
		setServiceAccountName(nsReconcilerName3),
		gceNodeMutator(nsReconcilerName3),
	)
	wantDeployments[core.IDOf(repoDeployment3)] = repoDeployment3
	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Resources successfully updated")

	// Test garbage collecting RoleBinding after all RepoSyncs are deleted
	if err := fakeClient.Delete(ctx, rs1); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName1); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Subject for rs1 is removed from RoleBinding.Subjects
	roleBinding1.Subjects = updateSubjects(roleBinding1.Subjects, nsReconcilerName)
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}

	if err := fakeClient.Delete(ctx, rs2); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName2); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Subject for rs2 is removed from RoleBinding.Subjects
	roleBinding2.Subjects = updateSubjects(roleBinding2.Subjects, nsReconcilerName2)
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}

	if err := fakeClient.Delete(ctx, rs3); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName3); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// roleBinding2 is deleted because there are no more RepoSyncs in the namespace.
	if err := validateResourceDeleted(roleBinding2, fakeClient); err != nil {
		t.Error(err)
	}
	delete(wantRoleBindings, core.IDOf(roleBinding2))

	if err := fakeClient.Delete(ctx, rs4); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName4); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Subject for rs4 is removed from RoleBinding.Subjects
	roleBinding1.Subjects = updateSubjects(roleBinding1.Subjects, nsReconcilerName4)
	if err := validateRoleBindings(wantRoleBindings, fakeClient); err != nil {
		t.Error(err)
	}

	if err := fakeClient.Delete(ctx, rs5); err != nil {
		t.Fatalf("failed to delete the root sync request, got error: %v, want error: nil", err)
	}
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName5); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}
	// Verify the RoleBinding is deleted after all RepoSyncs are deleted in the namespace.
	if err := validateResourceDeleted(roleBinding1, fakeClient); err != nil {
		t.Error(err)
	}
}

func addOwnerRefsForRootSync(reconcilerName string, cm *corev1.ConfigMap) {
	if strings.HasPrefix(reconcilerName, reconciler.RootReconcilerPrefix) {
		rsName := strings.TrimPrefix(reconcilerName, reconciler.RootReconcilerPrefix)
		if rsName == "" {
			rsName = configsync.RootSyncName
		} else {
			rsName = strings.TrimPrefix(rsName, "-")
		}
		cm.OwnerReferences = []metav1.OwnerReference{
			ownerReference(kinds.RootSyncV1Beta1().Kind, rsName, ""),
		}
	}
}

func addConfigMaps(m1, m2 map[core.ID]*corev1.ConfigMap) {
	for k, v := range m2 {
		m1[k] = v
	}
}

func buildWantConfigMaps(ctx context.Context, scope declared.Scope, syncName, reconcilerName string, git *v1beta1.Git, override *v1beta1.OverrideSpec) map[core.ID]*corev1.ConfigMap {
	gitSyncCMID, gitSyncCM := gitSyncConfigMap(ctx, reconcilerName, git, override)
	hydrationCMID, hydrationCM := hydrationConfigMap(scope, reconcilerName, git)
	reconcilerCMID, reconcilerCM := reconcilerConfigMap(scope, syncName, reconcilerName, git)
	sourceFormatCMID, sourceFormatCM := sourceFormatConfigMap(reconcilerName)
	cms := map[core.ID]*corev1.ConfigMap{
		gitSyncCMID:    gitSyncCM,
		hydrationCMID:  hydrationCM,
		reconcilerCMID: reconcilerCM,
	}
	if strings.HasPrefix(reconcilerName, reconciler.RootReconcilerPrefix) {
		cms[sourceFormatCMID] = sourceFormatCM
	}
	return cms
}

func gitSyncConfigMap(ctx context.Context, reconcilerName string, git *v1beta1.Git, override *v1beta1.OverrideSpec) (core.ID, *corev1.ConfigMap) {
	gitOptions := options{
		ref:        git.Revision,
		branch:     git.Branch,
		repo:       git.Repo,
		secretType: git.Auth,
		period:     configsync.DefaultPeriodSecs,
		proxy:      git.Proxy,
	}
	if git.NoSSLVerify {
		gitOptions.noSSLVerify = git.NoSSLVerify
	}
	if override.GitSyncDepth != nil {
		gitOptions.depth = override.GitSyncDepth
	}

	cm := configMapWithData(
		configsync.ControllerNamespace,
		ReconcilerResourceName(reconcilerName, reconcilermanager.GitSync),
		gitSyncData(ctx, gitOptions),
	)
	addOwnerRefsForRootSync(reconcilerName, cm)
	return core.IDOf(cm), cm
}

func hydrationConfigMap(scope declared.Scope, reconcilerName string, git *v1beta1.Git) (core.ID, *corev1.ConfigMap) {
	cm := configMapWithData(
		configsync.ControllerNamespace,
		ReconcilerResourceName(reconcilerName, reconcilermanager.HydrationController),
		hydrationData(git, scope, reconcilerName, pollingPeriod))
	addOwnerRefsForRootSync(reconcilerName, cm)
	return core.IDOf(cm), cm
}

func reconcilerConfigMap(scope declared.Scope, syncName, reconcilerName string, git *v1beta1.Git) (core.ID, *corev1.ConfigMap) {
	cm := configMapWithData(
		configsync.ControllerNamespace,
		ReconcilerResourceName(reconcilerName, reconcilermanager.Reconciler),
		reconcilerData(testCluster, syncName, reconcilerName, scope, git, pollingPeriod))
	addOwnerRefsForRootSync(reconcilerName, cm)
	return core.IDOf(cm), cm
}

func sourceFormatConfigMap(reconcilerName string) (core.ID, *corev1.ConfigMap) {
	cm := configMapWithData(
		configsync.ControllerNamespace,
		ReconcilerResourceName(reconcilerName, reconcilermanager.SourceFormat),
		sourceFormatData(""))
	addOwnerRefsForRootSync(reconcilerName, cm)
	return core.IDOf(cm), cm
}

func validateConfigMaps(wants map[core.ID]*corev1.ConfigMap, fakeClient *syncerFake.Client) error {
	for id, want := range wants {
		gotCoreObject := fakeClient.Objects[id]
		got := gotCoreObject.(*corev1.ConfigMap)
		if diff := cmp.Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
			return errors.Errorf("ConfigMap[%s/%s] diff: %s", got.Namespace, got.Name, diff)
		}
	}
	return nil
}

func validateServiceAccounts(wants map[core.ID]*corev1.ServiceAccount, fakeClient *syncerFake.Client) error {
	for id, want := range wants {
		gotCoreObject := fakeClient.Objects[id]
		got := gotCoreObject.(*corev1.ServiceAccount)
		if diff := cmp.Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
			return errors.Errorf("ServiceAccount[%s/%s] diff: %s", got.Namespace, got.Name, diff)
		}
	}
	return nil
}

func validateRoleBindings(wants map[core.ID]*rbacv1.RoleBinding, fakeClient *syncerFake.Client) error {
	for id, want := range wants {
		gotCoreObject := fakeClient.Objects[id]
		got := gotCoreObject.(*rbacv1.RoleBinding)
		if len(want.Subjects) != len(got.Subjects) {
			return errors.Errorf("RoleBinding[%s/%s] has unexpected number of subjects, expected %d, got %d",
				got.Namespace, got.Name, len(want.Subjects), len(got.Subjects))
		}
		for _, ws := range want.Subjects {
			for _, gs := range got.Subjects {
				if ws.Namespace == gs.Namespace && ws.Name == gs.Name {
					if !reflect.DeepEqual(ws, gs) {
						return errors.Errorf("RoleBinding[%s/%s] has unexpected subject, expected %v, got %v", got.Namespace, got.Name, ws, gs)
					}
				}
			}
		}
		got.Subjects = want.Subjects
		if diff := cmp.Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
			return errors.Errorf("RoleBinding[%s/%s] diff: %s", got.Namespace, got.Name, diff)
		}
	}
	return nil
}

func validateClusterRoleBinding(want *rbacv1.ClusterRoleBinding, fakeClient *syncerFake.Client) error {
	gotCoreObject := fakeClient.Objects[core.IDOf(want)]
	got := gotCoreObject.(*rbacv1.ClusterRoleBinding)
	if len(want.Subjects) != len(got.Subjects) {
		return errors.Errorf("ClusterRoleBinding[%s/%s] has unexpected number of subjects, expected %d, got %d",
			got.Namespace, got.Name, len(want.Subjects), len(got.Subjects))
	}
	for _, ws := range want.Subjects {
		for _, gs := range got.Subjects {
			if ws.Namespace == gs.Namespace && ws.Name == gs.Name {
				if !reflect.DeepEqual(ws, gs) {
					return errors.Errorf("ClusterRoleBinding[%s/%s] has unexpected subject, expected %v, got %v", got.Namespace, got.Name, ws, gs)
				}
			}
		}
	}
	got.Subjects = want.Subjects
	if diff := cmp.Diff(got, want, cmpopts.EquateEmpty()); diff != "" {
		return errors.Errorf("ClusterRoleBinding[%s/%s] diff: %s", got.Namespace, got.Name, diff)
	}
	return nil
}

// validateDeployments validates that important fields in the `wants` deployments match those same fields in the deployments found in the fakeClient
func validateDeployments(wants map[core.ID]*appsv1.Deployment, fakeClient *syncerFake.Client) error {
	for id, want := range wants {
		gotCoreObject := fakeClient.Objects[id]
		got := gotCoreObject.(*appsv1.Deployment)

		// Compare Deployment Annotations
		if diff := cmp.Diff(want.Annotations, got.Annotations); diff != "" {
			return errors.Errorf("Unexpected Deployment Annotations found for %q. Diff: %v", id, diff)
		}

		// Compare Deployment Template Annotations.
		if diff := cmp.Diff(want.Spec.Template.Annotations, got.Spec.Template.Annotations); diff != "" {
			return errors.Errorf("Unexpected Template Annotations found for %q. Diff: %v", id, diff)
		}

		// Compare ServiceAccountName.
		if diff := cmp.Diff(want.Spec.Template.Spec.ServiceAccountName, got.Spec.Template.Spec.ServiceAccountName); diff != "" {
			return errors.Errorf("Unexpected ServiceAccountName for %q. Diff: %v", id, diff)
		}

		// Compare Replicas
		if *want.Spec.Replicas != *got.Spec.Replicas {
			return errors.Errorf("Unexpected Replicas for %q. want %d, got %d", id, *want.Spec.Replicas, *got.Spec.Replicas)
		}

		// Compare Containers.
		if len(want.Spec.Template.Spec.Containers) != len(got.Spec.Template.Spec.Containers) {
			return errors.Errorf("Unexpected Container count for %q, want %d, got %d", id,
				len(want.Spec.Template.Spec.Containers), len(got.Spec.Template.Spec.Containers))
		}
		for _, i := range want.Spec.Template.Spec.Containers {
			for _, j := range got.Spec.Template.Spec.Containers {
				if i.Name == j.Name {
					// Compare EnvFrom fields in the container.
					if diff := cmp.Diff(i.EnvFrom, j.EnvFrom,
						cmpopts.SortSlices(func(x, y corev1.EnvFromSource) bool { return x.ConfigMapRef.Name < y.ConfigMapRef.Name })); diff != "" {
						return errors.Errorf("Unexpected configMapRef found for %q, diff %s", id, diff)
					}
					// Compare VolumeMount fields in the container.
					if diff := cmp.Diff(i.VolumeMounts, j.VolumeMounts,
						cmpopts.SortSlices(func(x, y corev1.VolumeMount) bool { return x.Name < y.Name })); diff != "" {
						return errors.Errorf("Unexpected volumeMount found for %q, diff %s", id, diff)
					}

					// Compare Env fields in the container.
					if diff := cmp.Diff(i.Env, j.Env,
						cmpopts.SortSlices(func(x, y corev1.EnvVar) bool { return x.Name < y.Name })); diff != "" {
						return errors.Errorf("Unexpected EnvVar found for %q, diff %s", id, diff)
					}

					// Compare Resources fields in the container.
					if diff := cmp.Diff(i.Resources, j.Resources); diff != "" {
						return errors.Errorf("Unexpected resources found for the %q container of %q, diff %s", i.Name, id, diff)
					}
				}
			}
		}

		// Compare Volumes
		if len(want.Spec.Template.Spec.Volumes) != len(got.Spec.Template.Spec.Volumes) {
			return errors.Errorf("Unexpected Volume count for %q, want %d, got %d", id,
				len(want.Spec.Template.Spec.Volumes), len(got.Spec.Template.Spec.Volumes))
		}
		for _, wantVolume := range want.Spec.Template.Spec.Volumes {
			for _, gotVolume := range got.Spec.Template.Spec.Volumes {
				if wantVolume.Name == gotVolume.Name {
					// Compare VolumeSource
					if !reflect.DeepEqual(wantVolume.VolumeSource, gotVolume.VolumeSource) {
						return errors.Errorf("Unexpected volume source for volume %s of %q, want %v, got %v",
							wantVolume.Name, id, wantVolume.VolumeSource, gotVolume.VolumeSource)
					}
				}
			}
		}
	}
	return nil
}

func validateResourceDeleted(resource client.Object, fakeClient *syncerFake.Client) error {
	if _, found := fakeClient.Objects[core.IDOf(resource)]; found {
		return errors.Errorf("resource %s still exists", core.IDOf(resource))
	}
	return nil
}

func updateSubjects(subjects []rbacv1.Subject, name string) []rbacv1.Subject {
	var result []rbacv1.Subject
	for _, s := range subjects {
		if s.Namespace != configsync.ControllerNamespace || s.Name != name {
			result = append(result, s)
		}
	}
	return result
}

func namespacedName(name, namespace string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func repoSyncDeployment(reconcilerName string, muts ...depMutator) *appsv1.Deployment {
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
