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

	gcpSAEmail = "config-sync@cs-project.iam.gserviceaccount.com"

	pollingPeriod = "50ms"

	// Hash of all configmap.data created by Namespace Reconciler.
	nsAnnotation      = "7c45f159e7c8005a792dcb402c078957"
	nsProxyAnnotation = "41b1ab0e5dbdeaf735a6e9f6740509c8"
	// Updated hash of all configmap.data updated by Namespace Reconciler.
	nsUpdatedAnnotation = "ad9eec9d09067c7aa5c339b3cef083f3"

	nsUpdatedAnnotationOverrideGitSyncDepth     = "7b357148ed82ae58c36940b25a563547"
	nsUpdatedAnnotationOverrideGitSyncDepthZero = "cb17525173bc95ea9cf3d067769fceb0"

	nsAnnotationGCENode        = "1e0a718052edc00039f6acc3738a02ae"
	nsUpdatedAnnotationGCENode = "4a5db0cdb29526ef77b8d3d9e3a18c06"
	nsAnnotationNone           = "551985d0f09594c9f82527a4ae3770d8"

	nsDeploymentGCENodeChecksum        = "d3a88860c72e5b39c38096c2dc657e85"
	nsDeploymentSecretChecksum         = "cc1e15e0abab9cd7e9ee92fd00340f65"
	nsDeploymentProxyChecksum          = "d6c4f21f2460ad88441a31376a99e761"
	nsDeploymentSecretUpdatedChecksum  = "9025b2dba5936ba6eb9aecbfcaa8008c"
	nsDeploymentGCENodeUpdatedChecksum = "c38f58f74e8a658f18b6bcb5005a4d4b"
	nsDeploymentNoneChecksum           = "a4aee26433917ddc4cce86d27b6f33cd"

	// Checksums of the Deployment whose container resource limits are updated
	nsDeploymentResourceLimitsChecksum                         = "94961815c1490fa2a0fd52f05b98dd2b"
	nsDeploymentReconcilerLimitsChecksum                       = "51ef264dfa236523ffa4b22fee6983de"
	nsDeploymentGitSyncMemLimitsChecksum                       = "b3a970a2d3c763b3da65a7033604b2f7"
	nsDeploymentReconcilerCPULimitsAndGitSyncMemLimitsChecksum = "a48562544a396572b3a96298e830757e"

	nsDeploymentSecretOverrideGitSyncDepthChecksum     = "63792a36e4f7ed8a81b71306a0f66724"
	nsDeploymentSecretOverrideGitSyncDepthZeroChecksum = "436570a6448441e799e98cea390de5b5"
)

// Set in init.
var filesystemPollingPeriod time.Duration

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

func TestRepoSyncReconcilerCreateAndUpdateRepoSyncWithOverride(t *testing.T) {
	// Mock out parseDeployment for testing.
	parseDeployment = parsedDeployment

	overrideReconcilerAndGitSyncResourceLimits := []v1alpha1.ContainerResourcesSpec{
		{
			ContainerName: "reconciler",
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: "git-sync",
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
			gitSyncData(options{
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
			secretMutator(nsDeploymentResourceLimitsChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			ContainerName: "reconciler",
			CPULimit:      resource.MustParse("1.2"),
		},
		{
			ContainerName: "git-sync",
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
			secretMutator(nsDeploymentReconcilerCPULimitsAndGitSyncMemLimitsChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			secretMutator(nsDeploymentSecretChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
	}

	if err := validateDeployments(wantDeployments, fakeClient); err != nil {
		t.Errorf("Deployment validation failed. err: %v", err)
	}
	t.Log("ConfigMap and Deployment successfully updated")
}

func TestRepoSyncReconcilerUpdateRepoSyncWithOverride(t *testing.T) {
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
			gitSyncData(options{
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
			secretMutator(nsDeploymentSecretChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			ContainerName: "reconciler",
			CPULimit:      resource.MustParse("1"),
			MemoryLimit:   resource.MustParse("1Gi"),
		},
		{
			ContainerName: "git-sync",
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
			secretMutator(nsDeploymentResourceLimitsChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			ContainerName: "reconciler",
			CPULimit:      resource.MustParse("2"),
			MemoryLimit:   resource.MustParse("2Gi"),
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
			secretMutator(nsDeploymentReconcilerLimitsChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			ContainerName: "git-sync",
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
			secretMutator(nsDeploymentGitSyncMemLimitsChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			secretMutator(nsDeploymentSecretChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
		),
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
			gitSyncData(options{
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
			secretMutator(nsDeploymentSecretOverrideGitSyncDepthChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			gitSyncData(options{
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
			secretMutator(nsDeploymentSecretChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			gitSyncData(options{
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
			secretMutator(nsDeploymentSecretOverrideGitSyncDepthChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			gitSyncData(options{
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
			secretMutator(nsDeploymentSecretOverrideGitSyncDepthZeroChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncSSHKey),
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
			gitSyncData(options{
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(options{
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.Reconciler),
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
			gitSyncData(options{
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
			RepoSyncResourceName(reposyncReqNamespace, reconcilermanager.GitSync),
			gitSyncData(options{
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
			gitSyncData(options{
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

func TestRepoSyncWithProxy(t *testing.T) {
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
			gitSyncData(options{
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
			setAnnotations(deploymentAnnotation(nsProxyAnnotation)),
			setServiceAccountName(reconciler.RepoSyncName(rs.Namespace)),
			secretMutator(nsDeploymentProxyChecksum, nsReconcilerName, nsReconcilerName+"-"+reposyncCookie),
			envVarMutator("HTTPS_PROXY", nsReconcilerName+"-"+reposyncCookie),
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
	for _, mut := range muts {
		mut(dep)
	}
	return dep
}
