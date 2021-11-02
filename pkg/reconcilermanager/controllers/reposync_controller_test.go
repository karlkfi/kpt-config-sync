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
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
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
	reposyncCookie       = "cookie"
	reposyncCluster      = "abc-123"

	secretName = "git-creds"

	gcpSAEmail = "config-sync@cs-project.iam.gserviceaccount.com"

	pollingPeriod = "50ms"

	// Hash of all configmap.data created by Namespace Reconciler.
	nsAnnotation                = "59152c00ed0f7cfcabe2da006293c638"
	nsProxyCookiefileAnnotation = "e8411037bea6bc20cc4011362c787129"
	nsProxyTokenAnnotation      = "096755b105668b37c6dbb0abdfc5af99"
	// Updated hash of all configmap.data updated by Namespace Reconciler.
	nsUpdatedAnnotation = "de29666acd0334d11b470903d42cfc38"

	nsUpdatedAnnotationOverrideGitSyncDepth     = "54dd1b6b43478af70ff5af41d278c988"
	nsUpdatedAnnotationOverrideGitSyncDepthZero = "b60f026202d1c059004abd839806efc1"

	nsUpdatedAnnotationNoSSLVerify = "d6a2b0e3a61735a005b4649d47fd97fd"

	nsAnnotationGCENode        = "1ae459d6e48a4b08514475ed0cdddecb"
	nsUpdatedAnnotationGCENode = "801faefb64294bf651f77ba7736fab17"
	nsAnnotationNone           = "096755b105668b37c6dbb0abdfc5af99"
)

// Set in init.
var filesystemPollingPeriod time.Duration
var hydrationPollingPeriod time.Duration

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
		glog.Exitf("failed to parse polling period: %q, got error: %v, want error: nil", pollingPeriod, err)
	}
	hydrationPollingPeriod = filesystemPollingPeriod
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

func reposyncOverrideResourceLimits(containers []v1alpha1.ContainerResourcesSpec) func(sync *v1alpha1.RepoSync) {
	return func(sync *v1alpha1.RepoSync) {
		sync.Spec.Override = v1alpha1.OverrideSpec{
			Resources: containers,
		}
	}
}

func reposyncOverrideGitSyncDepth(depth int64) func(*v1alpha1.RepoSync) {
	return func(rs *v1alpha1.RepoSync) {
		rs.Spec.Override.GitSyncDepth = &depth
	}
}

func reposyncNoSSLVerify() func(*v1alpha1.RepoSync) {
	return func(rs *v1alpha1.RepoSync) {
		rs.Spec.NoSSLVerify = true
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
		reposyncCluster,
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

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth),
		reposyncSecretRef(reposyncSSHKey), reposyncOverrideResourceLimits(overrideReconcilerAndGitSyncResourceLimits))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
			containerResourceLimitsMutator(overrideReconcilerCPULimitsAndGitSyncMemLimits),
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

	// Clear rs.Spec.Override
	rs.Spec.Override = v1alpha1.OverrideSpec{}
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestUpdateNamespaceReconcilerWithOverride(t *testing.T) {
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

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
			containerResourceLimitsMutator(overrideReconcilerAndGitSyncResourceLimits),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestCreateAndUpdateNamespaceReconcilerWithAutopilot(t *testing.T) {
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

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth),
		reposyncSecretRef(reposyncSSHKey), reposyncOverrideResourceLimits(overrideReconcilerAndGitSyncResourceLimits))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)
	isAutopilotCluster := true
	testReconciler.isAutopilotCluster = &isAutopilotCluster

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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

	// Test Autopilot ignores the resource requirements override.
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

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("Deployment successfully remained unchanged")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1alpha1.OverrideSpec{}

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

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)
	isAutopilotCluster := true
	testReconciler.isAutopilotCluster = &isAutopilotCluster

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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

	// Test Autopilot ignores resource requirements override.
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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully remained unchanged")

	// Clear rs.Spec.Override
	rs.Spec.Override = v1alpha1.OverrideSpec{}
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully remained unchanged")
}

func TestRepoSyncCreateWithNoSSLVerify(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey), reposyncNoSSLVerify())
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMapNoSSLVerify := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:         gitRevision,
				branch:      branch,
				repo:        reposyncRepo,
				secretType:  "ssh",
				period:      configsync.DefaultPeriodSecs,
				proxy:       rs.Spec.Proxy,
				noSSLVerify: rs.Spec.NoSSLVerify,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeploymentsNoSSLVerify := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationNoSSLVerify)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapNoSSLVerify {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsNoSSLVerify, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRepoSyncUpdateNoSSLVerify(t *testing.T) {
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

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:         gitRevision,
				branch:      branch,
				repo:        reposyncRepo,
				secretType:  "ssh",
				period:      configsync.DefaultPeriodSecs,
				proxy:       rs.Spec.Proxy,
				noSSLVerify: rs.Spec.NoSSLVerify,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMapNoSSLVerify := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:         gitRevision,
				branch:      branch,
				repo:        reposyncRepo,
				secretType:  "ssh",
				period:      configsync.DefaultPeriodSecs,
				proxy:       rs.Spec.Proxy,
				noSSLVerify: rs.Spec.NoSSLVerify,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeploymentsNoSSLVerify := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationNoSSLVerify)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapNoSSLVerify {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsNoSSLVerify, fakeClient); err != nil {
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

func TestRepoSyncCreateWithOverrideGitSyncDepth(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(auth), reposyncSecretRef(reposyncSSHKey), reposyncOverrideGitSyncDepth(5))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObj(t, reposyncSSHKey, secretAuth, core.Namespace(reposyncReqNamespace)))
	nsReconcilerName := reconciler.RepoSyncName(reposyncReqNamespace)

	// Test creating Configmaps and Deployment resources.
	ctx := context.Background()
	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMapOverrideGitSyncDepth := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
				depth:      rs.Spec.Override.GitSyncDepth,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeploymentsOverrideGitSyncDepth := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationOverrideGitSyncDepth)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapOverrideGitSyncDepth {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
		}
	}

	if err := validateDeployments(wantDeploymentsOverrideGitSyncDepth, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully created")
}

func TestRepoSyncUpdateOverrideGitSyncDepth(t *testing.T) {
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

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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

	// Test overriding the git sync depth to a positive value
	var depth int64 = 5
	rs.Spec.Override.GitSyncDepth = &depth
	if err := fakeClient.Update(ctx, rs); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMapOverrideGitSyncDepth := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
				depth:      rs.Spec.Override.GitSyncDepth,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeploymentsOverrideGitSyncDepth := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationOverrideGitSyncDepth)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapOverrideGitSyncDepth {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
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
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(ctx, reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMapOverrideGitSyncDepthZero := []*corev1.ConfigMap{
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
				depth:      rs.Spec.Override.GitSyncDepth,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeploymentsOverrideGitSyncDepthZero := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationOverrideGitSyncDepthZero)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	for _, cm := range wantConfigMapOverrideGitSyncDepthZero {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Config Map diff %s", diff)
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitUpdatedRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: "ssh",
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretGCENode))
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
			gceNodeMutator(nsReconcilerName),
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
			gceNodeMutator(nsReconcilerName),
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

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretGCPServiceAccount), reposyncGCPSAEmail(gcpSAEmail))
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: configsync.GitSecretGCPServiceAccount,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsReconcilerName),
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitUpdatedRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: configsync.GitSecretGCPServiceAccount,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantServiceAccount = fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments = []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsUpdatedAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsReconcilerName),
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

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretGCPServiceAccount), reposyncGCPSAEmail(gcpSAEmail))
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: configsync.GitSecretGCPServiceAccount,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
			reconcilerData(reposyncCluster, reposyncReqNamespace, &rs.Spec.Git, pollingPeriod),
		),
	}

	wantServiceAccount := fake.ServiceAccountObject(
		nsReconcilerName,
		core.Namespace(v1.NSConfigManagementSystem),
		core.Annotation(GCPSAAnnotationKey, rs.Spec.GCPServiceAccountEmail),
	)

	wantDeployments := []*appsv1.Deployment{
		repoSyncDeployment(
			rs,
			setAnnotations(deploymentAnnotation(nsAnnotationGCENode)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			gceNodeMutator(nsReconcilerName),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			noneMutator(nsReconcilerName),
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
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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

func TestRepoSyncCookiefileWithProxy(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretCookieFile), reposyncSecretRef(reposyncCookie))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	fakeClient, testReconciler := setupNSReconciler(t, rs, secretObjWithProxy(t, reposyncCookie, "cookie_file", core.Namespace(reposyncReqNamespace)))
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: configsync.GitSecretCookieFile,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
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
			setAnnotations(deploymentAnnotation(nsProxyCookiefileAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+reposyncCookie),
			envVarMutator("HTTPS_PROXY", nsReconcilerName+"-"+reposyncCookie, "https_proxy"),
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
}

func TestRepoSyncTokenWithProxy(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	rs := repoSync(reposyncRef(gitRevision), reposyncBranch(branch), reposyncSecretType(configsync.GitSecretToken), reposyncSecretRef(secretName))
	reqNamespacedName := namespacedName(reposyncCRName, reposyncReqNamespace)
	secret := secretObjWithProxy(t, secretName, GitSecretConfigKeyToken, core.Namespace(reposyncReqNamespace))
	secret.Data[GitSecretConfigKeyTokenUsername] = []byte("test-user")
	fakeClient, testReconciler := setupNSReconciler(t, rs, secret)

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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(ctx, options{
				ref:        gitRevision,
				branch:     branch,
				repo:       reposyncRepo,
				secretType: configsync.GitSecretToken,
				period:     configsync.DefaultPeriodSecs,
				proxy:      rs.Spec.Proxy,
			}),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.HydrationController),
			hydrationData(&rs.Spec.Git, reposyncReqNamespace, pollingPeriod),
		),
		configMapWithData(
			v1.NSConfigManagementSystem,
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
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
			setAnnotations(deploymentAnnotation(nsProxyTokenAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsReconcilerName, nsReconcilerName+"-"+secretName),
			envVarMutator("HTTPS_PROXY", nsReconcilerName+"-"+secretName, "https_proxy"),
			envVarMutator(gitSyncName, nsReconcilerName+"-"+secretName, GitSecretConfigKeyTokenUsername),
			envVarMutator(gitSyncPassword, nsReconcilerName+"-"+secretName, GitSecretConfigKeyToken),
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

					// Compare Env fields in the container.
					if diff := cmp.Diff(i.Env, j.Env,
						cmpopts.SortSlices(func(x, y corev1.EnvVar) bool { return x.Name < y.Name })); diff != "" {
						return errors.Errorf("Unexpected EnvVar found, diff %s", diff)
					}

					// Compare Resources fields in the container.
					if diff := cmp.Diff(i.Resources, j.Resources); diff != "" {
						return errors.Errorf("Unexpected resources found for the %q container, diff %s", i.Name, diff)
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
	dep.Annotations = nil
	for _, mut := range muts {
		mut(dep)
	}
	return dep
}
