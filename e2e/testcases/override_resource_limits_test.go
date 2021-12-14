package e2e

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/metrics"
	ocmetrics "github.com/google/nomos/pkg/metrics"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/reconcilermanager"
	"github.com/google/nomos/pkg/reconcilermanager/controllers"
	"github.com/google/nomos/pkg/testing/fake"
	"go.opencensus.io/tag"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

func defaultResourceLimits(nt *nomostest.NT) (corev1.ResourceList, corev1.ResourceList) {
	path := "../../manifests/templates/reconciler-manager-configmap.yaml"
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		nt.T.Fatalf("Failed to read file (%s): %v", path, err)
	}

	var cm corev1.ConfigMap
	if err := yaml.Unmarshal(contents, &cm); err != nil {
		nt.T.Fatalf("Failed to parse the ConfigMap object in %s: %v", path, err)
	}

	key := "deployment.yaml"
	deployContents, ok := cm.Data[key]
	if !ok {
		nt.T.Fatalf("The `data` field of the ConfigMap object in %s does not include the %q key", path, key)
	}

	var deploy appsv1.Deployment
	if err := yaml.Unmarshal([]byte(deployContents), &deploy); err != nil {
		nt.T.Fatalf("Failed to parse the Deployment object in the `data` field of the ConfigMap object in %s: %v", path, err)
	}

	var reconcilerResourceLimits, gitSyncResourceLimits corev1.ResourceList
	for _, container := range deploy.Spec.Template.Spec.Containers {
		if container.Name == reconcilermanager.Reconciler || container.Name == reconcilermanager.GitSync {
			if container.Resources.Limits.Cpu().IsZero() || container.Resources.Limits.Memory().IsZero() ||
				container.Resources.Requests.Cpu().IsZero() || container.Resources.Requests.Memory().IsZero() {
				nt.T.Fatalf("The %s container in %s should define CPU/memory limits and requests", container.Name, path)
			}
		}
		if container.Name == reconcilermanager.Reconciler {
			reconcilerResourceLimits = container.Resources.Limits
		}
		if container.Name == reconcilermanager.GitSync {
			gitSyncResourceLimits = container.Resources.Limits
		}
	}
	return reconcilerResourceLimits, gitSyncResourceLimits
}

func TestOverrideResourceLimitsV1Alpha1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.SkipAutopilotCluster, ntopts.NamespaceRepo(backendNamespace), ntopts.NamespaceRepo(frontendNamespace))
	nt.WaitForRepoSyncs()

	// Get the default CPU/memory limits of the reconciler container and the git-sync container
	reconcilerResourceLimits, gitSyncResourceLimits := defaultResourceLimits(nt)
	defaultReconcilerCPULimits, defaultReconcilerMemLimits := reconcilerResourceLimits[corev1.ResourceCPU], reconcilerResourceLimits[corev1.ResourceMemory]
	defaultGitSyncCPULimits, defaultGitSyncMemLimits := gitSyncResourceLimits[corev1.ResourceCPU], gitSyncResourceLimits[corev1.ResourceMemory]

	// Verify root-reconciler uses the default resource limits
	err := nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-backend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-frontend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
			return nt.ValidateMetricNotFound(ocmetrics.ResourceOverrideCountView.Name)
		})
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	backendRepo, exist := nt.NonRootRepos[backendNamespace]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}

	frontendRepo, exist := nt.NonRootRepos[frontendNamespace]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}

	rootSync := fake.RootSyncObject()
	repoSyncBackend := nomostest.RepoSyncObject(backendNamespace, nt.GitProvider.SyncURL(backendRepo.RemoteRepoName))
	repoSyncFrontend := nomostest.RepoSyncObject(frontendNamespace, nt.GitProvider.SyncURL(frontendRepo.RemoteRepoName))

	// Override the CPU/memory limits of the reconciler container of root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"resources": [{"containerName": "reconciler", "cpuLimit": "800m", "memoryLimit": "411Mi"}]}}}`)

	// Verify the reconciler container of root-reconciler uses the new resource limits, and the git-sync container uses the default resource limits.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
			nomostest.HasCorrectResourceLimits(resource.MustParse("800m"), resource.MustParse("411Mi"), defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-backend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-frontend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the CPU/memory limits of the reconciler container of ns-reconciler-backend
	repoSyncBackend.Spec.Override = v1alpha1.OverrideSpec{
		Resources: []v1alpha1.ContainerResourcesSpec{
			{
				ContainerName: "reconciler",
				CPULimit:      resource.MustParse("555m"),
				MemoryLimit:   resource.MustParse("555Mi"),
			},
			{
				ContainerName: "git-sync",
				CPULimit:      resource.MustParse("666m"),
				MemoryLimit:   resource.MustParse("666Mi"),
			},
		},
	}
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)

	// Override the CPU/memory limits of the reconciler container of ns-reconciler-frontend
	repoSyncFrontend.Spec.Override = v1alpha1.OverrideSpec{
		Resources: []v1alpha1.ContainerResourcesSpec{
			{
				ContainerName: "reconciler",
				CPULimit:      resource.MustParse("544m"),
				MemoryLimit:   resource.MustParse("544Mi"),
			},
			{
				ContainerName: "git-sync",
				CPULimit:      resource.MustParse("644m"),
				MemoryLimit:   resource.MustParse("644Mi"),
			},
		},
	}
	nt.Root.Add(nomostest.StructuredNSPath(frontendNamespace, nomostest.RepoSyncFileName), repoSyncFrontend)
	nt.Root.CommitAndPush("Update backend and frontend RepoSync resource limits")
	nt.WaitForRepoSyncs()

	// Verify the resource limits of root-reconciler are not affected by the resource limit change of ns-reconciler-backend and ns-reconciler-fronend
	err = nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("800m"), resource.MustParse("411Mi"), defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-backend uses the new resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("555m"), resource.MustParse("555Mi"), resource.MustParse("666m"), resource.MustParse("666Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-frontend uses the new resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the CPU limit of the git-sync container of root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"resources": [{"containerName": "git-sync", "cpuLimit": "333m"}]}}}`)

	// Verify the reconciler container root-reconciler uses the default resource limits, and the git-sync container uses the new resource limits.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
			nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, resource.MustParse("333m"), defaultGitSyncMemLimits))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-backend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("555m"), resource.MustParse("555Mi"), resource.MustParse("666m"), resource.MustParse("666Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-frontend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		if err := nt.ValidateResourceOverrideCount(string(controllers.RootReconcilerType), "git-sync", "cpu", 1); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCountMissingTags([]tag.Tag{
			{Key: metrics.KeyReconcilerType, Value: string(controllers.RootReconcilerType)},
			{Key: metrics.KeyContainer, Value: "reconciler"},
			{Key: metrics.KeyResourceType, Value: "memory"},
		}); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "cpu", 2); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "memory", 2); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "cpu", 2); err != nil {
			return err
		}
		return nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "memory", 2)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from the RootSync
	nt.MustMergePatch(rootSync, `{"spec": {"override": null}}`)

	// Verify root-reconciler uses the default resource limits
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
			nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-backend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("555m"), resource.MustParse("555Mi"), resource.MustParse("666m"), resource.MustParse("666Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-frontend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from repoSyncBackend
	repoSyncBackend.Spec.Override = v1alpha1.OverrideSpec{}
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Clear `spec.override` from repoSyncBackend")
	nt.WaitForRepoSyncs()

	// Verify ns-reconciler-backend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify root-reconciler uses the default resource limits
	err = nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-frontend are not affected by the resource limit change of ns-reconciler-backend
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		if err := nt.ValidateResourceOverrideCountMissingTags([]tag.Tag{
			{Key: metrics.KeyReconcilerType, Value: string(controllers.RootReconcilerType)},
		}); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "cpu", 1); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "memory", 1); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "cpu", 1); err != nil {
			return err
		}
		return nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "memory", 1)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from repoSyncFrontend
	repoSyncFrontend.Spec.Override = v1alpha1.OverrideSpec{}
	nt.Root.Add(nomostest.StructuredNSPath(frontendNamespace, nomostest.RepoSyncFileName), repoSyncFrontend)
	nt.Root.CommitAndPush("Clear `spec.override` from repoSyncFrontend")
	nt.WaitForRepoSyncs()

	// Verify ns-reconciler-frontend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
			return nt.ValidateMetricNotFound(ocmetrics.ResourceOverrideCountView.Name)
		})
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}

func TestOverrideResourceLimitsV1Beta1(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.SkipAutopilotCluster, ntopts.NamespaceRepo(backendNamespace), ntopts.NamespaceRepo(frontendNamespace))
	nt.WaitForRepoSyncs()

	// Get the default CPU/memory limits of the reconciler container and the git-sync container
	reconcilerResourceLimits, gitSyncResourceLimits := defaultResourceLimits(nt)
	defaultReconcilerCPULimits, defaultReconcilerMemLimits := reconcilerResourceLimits[corev1.ResourceCPU], reconcilerResourceLimits[corev1.ResourceMemory]
	defaultGitSyncCPULimits, defaultGitSyncMemLimits := gitSyncResourceLimits[corev1.ResourceCPU], gitSyncResourceLimits[corev1.ResourceMemory]

	// Verify root-reconciler uses the default resource limits
	err := nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-backend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-frontend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		return nt.ValidateMetricNotFound(ocmetrics.ResourceOverrideCountView.Name)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	backendRepo, exist := nt.NonRootRepos[backendNamespace]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}

	frontendRepo, exist := nt.NonRootRepos[frontendNamespace]
	if !exist {
		nt.T.Fatal("nonexistent repo")
	}

	rootSync := fake.RootSyncObjectV1Beta1()
	repoSyncBackend := nomostest.RepoSyncObjectV1Beta1(backendNamespace, nt.GitProvider.SyncURL(backendRepo.RemoteRepoName))
	repoSyncFrontend := nomostest.RepoSyncObjectV1Beta1(frontendNamespace, nt.GitProvider.SyncURL(frontendRepo.RemoteRepoName))

	// Override the CPU/memory limits of the reconciler container of root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"resources": [{"containerName": "reconciler", "cpuLimit": "800m", "memoryLimit": "411Mi"}]}}}`)

	// Verify the reconciler container of root-reconciler uses the new resource limits, and the git-sync container uses the default resource limits.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
			nomostest.HasCorrectResourceLimits(resource.MustParse("800m"), resource.MustParse("411Mi"), defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-backend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-frontend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the CPU/memory limits of the reconciler container of ns-reconciler-backend
	repoSyncBackend.Spec.Override = v1beta1.OverrideSpec{
		Resources: []v1beta1.ContainerResourcesSpec{
			{
				ContainerName: "reconciler",
				CPULimit:      resource.MustParse("555m"),
				MemoryLimit:   resource.MustParse("555Mi"),
			},
			{
				ContainerName: "git-sync",
				CPULimit:      resource.MustParse("666m"),
				MemoryLimit:   resource.MustParse("666Mi"),
			},
		},
	}
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)

	// Override the CPU/memory limits of the reconciler container of ns-reconciler-frontend
	repoSyncFrontend.Spec.Override = v1beta1.OverrideSpec{
		Resources: []v1beta1.ContainerResourcesSpec{
			{
				ContainerName: "reconciler",
				CPULimit:      resource.MustParse("544m"),
				MemoryLimit:   resource.MustParse("544Mi"),
			},
			{
				ContainerName: "git-sync",
				CPULimit:      resource.MustParse("644m"),
				MemoryLimit:   resource.MustParse("644Mi"),
			},
		},
	}
	nt.Root.Add(nomostest.StructuredNSPath(frontendNamespace, nomostest.RepoSyncFileName), repoSyncFrontend)
	nt.Root.CommitAndPush("Update backend and frontend RepoSync resource limits")
	nt.WaitForRepoSyncs()

	// Verify the resource limits of root-reconciler are not affected by the resource limit change of ns-reconciler-backend and ns-reconciler-fronend
	err = nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("800m"), resource.MustParse("411Mi"), defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-backend uses the new resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("555m"), resource.MustParse("555Mi"), resource.MustParse("666m"), resource.MustParse("666Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify ns-reconciler-frontend uses the new resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Override the CPU limit of the git-sync container of root-reconciler
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"resources": [{"containerName": "git-sync", "cpuLimit": "333m"}]}}}`)

	// Verify the reconciler container root-reconciler uses the default resource limits, and the git-sync container uses the new resource limits.
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
			nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, resource.MustParse("333m"), defaultGitSyncMemLimits))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-backend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("555m"), resource.MustParse("555Mi"), resource.MustParse("666m"), resource.MustParse("666Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-frontend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		if err := nt.ValidateResourceOverrideCount(string(controllers.RootReconcilerType), "git-sync", "cpu", 1); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCountMissingTags([]tag.Tag{
			{Key: metrics.KeyReconcilerType, Value: string(controllers.RootReconcilerType)},
			{Key: metrics.KeyContainer, Value: "reconciler"},
			{Key: metrics.KeyResourceType, Value: "memory"},
		}); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "cpu", 2); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "memory", 2); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "cpu", 2); err != nil {
			return err
		}
		return nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "memory", 2)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from the RootSync
	nt.MustMergePatch(rootSync, `{"spec": {"override": null}}`)

	// Verify root-reconciler uses the default resource limits
	_, err = nomostest.Retry(30*time.Second, func() error {
		return nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
			nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-backend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("555m"), resource.MustParse("555Mi"), resource.MustParse("666m"), resource.MustParse("666Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-frontend are not affected by the resource limit change of root-reconciler
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from repoSyncBackend
	repoSyncBackend.Spec.Override = v1beta1.OverrideSpec{}
	nt.Root.Add(nomostest.StructuredNSPath(backendNamespace, nomostest.RepoSyncFileName), repoSyncBackend)
	nt.Root.CommitAndPush("Clear `spec.override` from repoSyncBackend")
	nt.WaitForRepoSyncs()

	// Verify ns-reconciler-backend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(backendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify root-reconciler uses the default resource limits
	err = nt.Validate(reconciler.RootSyncName, v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	// Verify the resource limits of ns-reconciler-frontend are not affected by the resource limit change of ns-reconciler-backend
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(resource.MustParse("544m"), resource.MustParse("544Mi"), resource.MustParse("644m"), resource.MustParse("644Mi")))
	if err != nil {
		nt.T.Fatal(err)
	}

	err = nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
		if err := nt.ValidateResourceOverrideCountMissingTags([]tag.Tag{
			{Key: metrics.KeyReconcilerType, Value: string(controllers.RootReconcilerType)},
		}); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "cpu", 1); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "reconciler", "memory", 1); err != nil {
			return err
		}
		if err := nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "cpu", 1); err != nil {
			return err
		}
		return nt.ValidateResourceOverrideCount(string(controllers.NamespaceReconcilerType), "git-sync", "memory", 1)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Clear `spec.override` from repoSyncFrontend
	repoSyncFrontend.Spec.Override = v1beta1.OverrideSpec{}
	nt.Root.Add(nomostest.StructuredNSPath(frontendNamespace, nomostest.RepoSyncFileName), repoSyncFrontend)
	nt.Root.CommitAndPush("Clear `spec.override` from repoSyncFrontend")
	nt.WaitForRepoSyncs()

	// Verify ns-reconciler-frontend uses the default resource limits
	err = nt.Validate(reconciler.RepoSyncName(frontendNamespace), v1.NSConfigManagementSystem, &appsv1.Deployment{},
		nomostest.HasCorrectResourceLimits(defaultReconcilerCPULimits, defaultReconcilerMemLimits, defaultGitSyncCPULimits, defaultGitSyncMemLimits))
	if err != nil {
		nt.T.Fatal(err)
	}

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.ValidateMetrics(nomostest.SyncMetricsToLatestCommit(nt), func() error {
			return nt.ValidateMetricNotFound(ocmetrics.ResourceOverrideCountView.Name)
		})
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
