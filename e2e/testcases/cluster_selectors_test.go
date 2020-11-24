package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	prodClusterName         = "e2e-test-cluster"
	testClusterName         = "test-cluster-env-test"
	environmentLabelKey     = "environment"
	prodEnvironment         = "prod"
	testEnvironment         = "test"
	prodClusterSelectorName = "selector-env-prod"
	testClusterSelectorName = "selector-env-test"
	frontendNamespace       = "frontend"
	backendNamespace        = "backend"
	roleBindingName         = "bob-rolebinding"
	namespaceRepo           = "bookstore"
)

var (
	inlineProdClusterSelectorAnnotation = map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: prodClusterName}
	legacyTestClusterSelectorAnnotation = map[string]string{v1.LegacyClusterSelectorAnnotationKey: testClusterSelectorName}
)

func clusterObject(name, label, value string) *clusterregistry.Cluster {
	return fake.ClusterObject(core.Name(name), core.Label(label, value))
}

func clusterSelector(name, label, value string) *v1.ClusterSelector {
	cs := fake.ClusterSelectorObject(core.Name(name))
	cs.Spec.Selector.MatchLabels = map[string]string{label: value}
	return cs
}

func resourceQuota(name, pods string, annotations map[string]string) *corev1.ResourceQuota {
	rq := fake.ResourceQuotaObject(core.Name(name), core.Annotations(annotations))
	rq.Spec.Hard = map[corev1.ResourceName]resource.Quantity{corev1.ResourcePods: resource.MustParse(pods)}
	return rq
}

func roleBinding(name string, annotations map[string]string) *rbacv1.RoleBinding {
	rb := fake.RoleBindingObject(core.Name(name),
		core.Annotations(annotations))
	rb.Subjects = []rbacv1.Subject{{
		Kind: "User", Name: "bob@acme.com", APIGroup: rbacv1.GroupName,
	}}
	rb.RoleRef = rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "acme-admin",
		APIGroup: rbacv1.GroupName,
	}
	return rb
}

func namespaceObject(name string, annotations map[string]string) *corev1.Namespace {
	return fake.NamespaceObject(name, core.Annotations(annotations))
}

func TestTargetingDifferentResourceQuotasToDifferentClusters(t *testing.T) {
	nt := nomostest.New(t)
	configMapName := clusterNameConfigMapName(nt)

	nt.T.Log("Add test cluster, and cluster registry data")
	testCluster := clusterObject(testClusterName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/cluster-test.yaml", testCluster)
	testClusterSelector := clusterSelector(testClusterSelectorName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/clusterselector-test.yaml", testClusterSelector)
	nt.Root.CommitAndPush("Add test cluster and cluster registry data")

	t.Log("Add a valid cluster selector annotation to a resource quota")
	resourceQuotaName := "pod-quota"
	prodPodsQuota := "133"
	testPodsQuota := "266"
	rqInline := resourceQuota(resourceQuotaName, prodPodsQuota, inlineProdClusterSelectorAnnotation)
	rqLegacy := resourceQuota(resourceQuotaName, testPodsQuota, legacyTestClusterSelectorAnnotation)
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", frontendNamespace),
		namespaceObject(frontendNamespace, map[string]string{}))
	nt.Root.Add("acme/namespaces/eng/quota-inline.yaml", rqInline)
	nt.Root.Add("acme/namespaces/eng/quota-legacy.yaml", rqLegacy)
	nt.Root.CommitAndPush("Add a valid cluster selector annotation to a resource quota")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(resourceQuotaName, frontendNamespace, &corev1.ResourceQuota{}, resourceQuotaHasHardPods(prodPodsQuota)); err != nil {
		t.Fatal(err)
	}

	renameCluster(nt, configMapName, testClusterName)
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	nomostest.Wait(nt.T, "resource quota is changed to test pods", func() error {
		return nt.Validate(resourceQuotaName, frontendNamespace, &corev1.ResourceQuota{}, resourceQuotaHasHardPods(testPodsQuota))
	})

	renameCluster(nt, configMapName, prodClusterName)
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	nomostest.Wait(nt.T, "resource quota is changed to test pods", func() error {
		return nt.Validate(resourceQuotaName, frontendNamespace, &corev1.ResourceQuota{}, resourceQuotaHasHardPods(prodPodsQuota))
	})
}

func TestClusterSelectorOnObjects(t *testing.T) {
	nt := nomostest.New(t)

	configMapName := clusterNameConfigMapName(nt)

	t.Log("Add a valid cluster selector annotation to a role binding")
	rb := roleBinding(roleBindingName, inlineProdClusterSelectorAnnotation)
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a valid cluster selector annotation to a role binding")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	nt.T.Log("Add test cluster, and cluster registry data")
	testCluster := clusterObject(testClusterName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/cluster-test.yaml", testCluster)
	testClusterSelector := clusterSelector(testClusterSelectorName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/clusterselector-test.yaml", testClusterSelector)
	nt.Root.CommitAndPush("Add test cluster and cluster registry data")

	t.Log("Change cluster selector to match test cluster")
	rb.Annotations = legacyTestClusterSelectorAnnotation
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Change cluster selector to match test cluster")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	renameCluster(nt, configMapName, testClusterName)
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	nomostest.Wait(nt.T, "rolebinding reappears", func() error {
		return nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{})
	})

	t.Log("Revert cluster selector to match prod cluster")
	rb.Annotations = inlineProdClusterSelectorAnnotation
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Revert cluster selector to match prod cluster")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	renameCluster(nt, configMapName, prodClusterName)
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	nomostest.Wait(nt.T, "rolebinding reappears", func() error {
		return nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{})
	})
}

func TestClusterSelectorOnNamespaces(t *testing.T) {
	nt := nomostest.New(t)

	configMapName := clusterNameConfigMapName(nt)

	t.Log("Add a valid cluster selector annotation to a namespace")
	namespace := namespaceObject(backendNamespace, inlineProdClusterSelectorAnnotation)
	rb := roleBinding(roleBindingName, inlineProdClusterSelectorAnnotation)
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	nt.Root.Add("acme/namespaces/eng/backend/namespace.yaml", namespace)
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a valid cluster selector annotation to a namespace and a role binding")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(backendNamespace, "", &corev1.Namespace{}); err != nil {
		t.Fatal(err)
	}
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Add test cluster, and cluster registry data")
	testCluster := clusterObject(testClusterName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/cluster-test.yaml", testCluster)
	testClusterSelector := clusterSelector(testClusterSelectorName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/clusterselector-test.yaml", testClusterSelector)
	nt.Root.CommitAndPush("Add test cluster and cluster registry data")

	t.Log("Change namespace to match test cluster")
	namespace.Annotations = legacyTestClusterSelectorAnnotation
	nt.Root.Add("acme/namespaces/eng/backend/namespace.yaml", namespace)
	nt.Root.CommitAndPush("Change namespace to match test cluster")
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	// The validation failed intermittently only in Mono repo
	nomostest.Wait(nt.T, "namespace reappears", func() error {
		return nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{})
	})
	nomostest.WaitToTerminate(nt, kinds.Namespace(), backendNamespace, "")

	renameCluster(nt, configMapName, testClusterName)
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	nomostest.Wait(nt.T, "namespace reappears", func() error {
		return nt.Validate(backendNamespace, "", &corev1.Namespace{})
	})
	// bob-rolebinding won't reappear in the backend namespace as the cluster is inactive in the cluster-selector
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Updating bob-rolebinding to NOT have cluster-selector")
	rb.Annotations = map[string]string{}
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Update bob-rolebinding to NOT have cluster-selector")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Revert namespace to match prod cluster")
	namespace.Annotations = inlineProdClusterSelectorAnnotation
	nt.Root.Add("acme/namespaces/eng/backend/namespace.yaml", namespace)
	nt.Root.CommitAndPush("Revert namespace to match prod cluster")
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	// The validation failed intermittently only in Mono repo
	nomostest.Wait(nt.T, "namespace reappears", func() error {
		return nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{})
	})
	nomostest.WaitToTerminate(nt, kinds.Namespace(), backendNamespace, "")

	renameCluster(nt, configMapName, prodClusterName)
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	nomostest.Wait(nt.T, "namespace reappears", func() error {
		if err := nt.Validate(backendNamespace, "", &corev1.Namespace{}); err != nil {
			return err
		}
		return nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{})
	})
}

func TestObjectReactsToChangeInInlineClusterSelector(t *testing.T) {
	nt := nomostest.New(t)

	t.Log("Add a valid cluster selector annotation to a role binding")
	rb := roleBinding(roleBindingName, inlineProdClusterSelectorAnnotation)
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a valid cluster selector annotation to a role binding")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Modify the cluster selector to select an excluded cluster list")
	rb.Annotations = map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: "a, b, c"}
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Modify the cluster selector to select an excluded cluster list")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}
}

func TestObjectReactsToChangeInLegacyClusterSelector(t *testing.T) {
	nt := nomostest.New(t)

	nt.T.Log("Add prod cluster, and cluster registry data")
	prodCluster := clusterObject(prodClusterName, environmentLabelKey, prodEnvironment)
	nt.Root.Add("acme/clusterregistry/cluster-prod.yaml", prodCluster)
	prodClusterSelector := clusterSelector(prodClusterSelectorName, environmentLabelKey, prodEnvironment)
	nt.Root.Add("acme/clusterregistry/clusterselector-prod.yaml", prodClusterSelector)
	nt.Root.CommitAndPush("Add prod cluster and cluster registry data")

	t.Log("Add a valid cluster selector annotation to a role binding")
	rb := roleBinding(roleBindingName, map[string]string{v1.LegacyClusterSelectorAnnotationKey: prodClusterSelectorName})
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a valid cluster selector annotation to a role binding")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Modify the cluster selector to select a different environment")
	prodClusterWithADifferentSelector := clusterSelector(prodClusterSelectorName, environmentLabelKey, "other")
	nt.Root.Add("acme/clusterregistry/clusterselector-prod.yaml", prodClusterWithADifferentSelector)
	nt.Root.CommitAndPush("Modify the cluster selector to select a different environment")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}
}

func TestImporterIgnoresNonSelectedCustomResources(t *testing.T) {
	nt := nomostest.New(t)

	nt.T.Log("Add test cluster, and cluster registry data")
	testCluster := clusterObject(testClusterName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/cluster-test.yaml", testCluster)
	testClusterSelector := clusterSelector(testClusterSelectorName, environmentLabelKey, testEnvironment)
	nt.Root.Add("acme/clusterregistry/clusterselector-test.yaml", testClusterSelector)
	nt.Root.CommitAndPush("Add test cluster and cluster registry data")

	t.Log("Add CRs (not targeted to this cluster) without its CRD")
	cr := anvilCR("v1", "e2e-test-anvil", 10)
	cr.SetAnnotations(map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: testClusterSelectorName})
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	nt.Root.Add("acme/namespaces/eng/backend/anvil.yaml", cr)
	cr2 := anvilCR("v1", "e2e-test-anvil-2", 10)
	cr2.SetAnnotations(legacyTestClusterSelectorAnnotation)
	nt.Root.Add("acme/namespaces/eng/backend/anvil-2.yaml", cr2)
	nt.Root.CommitAndPush("Add a custom resource without its CRD")

	nt.WaitForRepoSyncs()
}

func TestInlineClusterSelectorOnNamespaceRepos(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.NamespaceRepo(namespaceRepo),
	)

	t.Log("Add a valid cluster selector annotation to a role binding")
	rb := roleBinding(roleBindingName, inlineProdClusterSelectorAnnotation)
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	nt.NonRootRepos[namespaceRepo].Add("acme/bob-rolebinding.yaml", rb)
	nt.NonRootRepos[namespaceRepo].CommitAndPush("Add a valid cluster selector annotation to a role binding")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, namespaceRepo, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Modify the cluster selector to select an excluded cluster list")
	rb.Annotations = map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: "a,b,,,c,d"}
	nt.NonRootRepos[namespaceRepo].Add("acme/bob-rolebinding.yaml", rb)
	nt.NonRootRepos[namespaceRepo].CommitAndPush("Modify the cluster selector to select an excluded cluster list")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, namespaceRepo, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}
}

func TestInlineClusterSelectorFormat(t *testing.T) {
	nt := nomostest.New(t)

	configMapName := clusterNameConfigMapName(nt)
	renameCluster(nt, configMapName, "")

	t.Log("Add a role binding without any cluster selectors")
	rb := roleBinding(roleBindingName, map[string]string{})
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a role binding without any cluster selectors")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Logf("Add a prod cluster selector to the role binding")
	rb.Annotations = inlineProdClusterSelectorAnnotation
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a prod cluster selector to the role binding")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	renameCluster(nt, configMapName, prodClusterName)
	nt.WaitForRepoSyncs()
	// TODO(b/175227055): ideally no need to retry after nt.WaitForRepoSyncs(), but the test failed intermittently without wait and retry.
	// More investigation is needed to figure out why resource isn't updated immediately when it is marked as 'synced'.
	nomostest.Wait(nt.T, "rolebinding reappears", func() error {
		return nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{})
	})

	t.Log("Add an empty cluster selector annotation to a role binding")
	rb.Annotations = map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: ""}
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add an empty cluster selector annotation to a role binding")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Add a cluster selector annotation to a role binding with a list of included clusters")
	rb.Annotations = map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: fmt.Sprintf("a,%s,b", prodClusterName)}
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a cluster selector annotation to a role binding with a list of included clusters")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Add a cluster selector annotation to a role binding with a list of excluded clusters")
	rb.Annotations = map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: "a,,b"}
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a cluster selector annotation to a role binding with a list of excluded clusters")
	nt.WaitForRepoSyncs()
	if err := nt.ValidateNotFound(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}

	t.Log("Add a cluster selector annotation to a role binding with a list of included clusters (with spaces)")
	rb.Annotations = map[string]string{v1alpha1.ClusterNameSelectorAnnotationKey: fmt.Sprintf("a , %s , b", prodClusterName)}
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add a cluster selector annotation to a role binding with a list of included clusters (with spaces)")
	nt.WaitForRepoSyncs()
	if err := nt.Validate(roleBindingName, backendNamespace, &rbacv1.RoleBinding{}); err != nil {
		t.Fatal(err)
	}
}

func TestClusterSelectorAnnotationConflicts(t *testing.T) {
	nt := nomostest.New(t)

	t.Log("Add both cluster selector annotations to a role binding")
	nt.Root.Add(
		fmt.Sprintf("acme/namespaces/eng/%s/namespace.yaml", backendNamespace),
		namespaceObject(backendNamespace, map[string]string{}))
	rb := roleBinding(roleBindingName, map[string]string{
		v1alpha1.ClusterNameSelectorAnnotationKey: prodClusterName,
		v1.LegacyClusterSelectorAnnotationKey:     prodClusterSelectorName,
	})
	nt.Root.Add("acme/namespaces/eng/backend/bob-rolebinding.yaml", rb)
	nt.Root.CommitAndPush("Add both cluster selector annotations to a role binding")
	if nt.MultiRepo {
		nt.WaitForRootSyncSourceErrorCode(selectors.ClusterSelectorAnnotationConflictErrorCode)
	} else {
		nt.WaitForRepoImportErrorCode(selectors.ClusterSelectorAnnotationConflictErrorCode)
	}
}

// renameCluster updates CLUSTER_NAME in the config map and restart the reconcilers.
func renameCluster(nt *nomostest.NT, configMapName, clusterName string) {
	nt.T.Logf("Change the cluster name to %q", clusterName)
	cm := &corev1.ConfigMap{}
	err := nt.Get(configMapName, configmanagement.ControllerNamespace, cm)
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.MustMergePatch(cm, fmt.Sprintf(`{"data":{"CLUSTER_NAME":"%s"}}`, clusterName))

	if nt.MultiRepo {
		deletePodByLabel(nt, "app", "reconciler-manager")
	} else {
		deletePodByLabel(nt, "app", "git-importer")
		deletePodByLabel(nt, "app", "monitor")
	}
}

// deletePodByLabel deletes pods that have the label and waits until new pods come up.
func deletePodByLabel(nt *nomostest.NT, label, value string) {
	oldPods := &corev1.PodList{}
	if err := nt.List(oldPods, client.InNamespace(configmanagement.ControllerNamespace), client.MatchingLabels{label: value}); err != nil {
		nt.T.Fatal(err)
	}
	if err := nt.DeleteAllOf(&corev1.Pod{}, client.InNamespace(configmanagement.ControllerNamespace), client.MatchingLabels{label: value}); err != nil {
		nt.T.Fatalf("Pod delete failed: %v", err)
	}
	nomostest.Wait(nt.T, "new pods come up", func() error {
		podList := &corev1.PodList{}
		if err := nt.List(podList, client.InNamespace(configmanagement.ControllerNamespace), client.MatchingLabels{label: value}); err != nil {
			nt.T.Fatal(err)
		}
		for _, newPod := range podList.Items {
			for _, oldPod := range oldPods.Items {
				if newPod.Name == oldPod.Name {
					return fmt.Errorf("old pod %s is still alive", oldPod.Name)
				}
			}
		}
		return nil
	}, nomostest.WaitTimeout(time.Minute))
}

// clusterNameConfigMapName returns the name of the ConfigMap that has the CLUSTER_NAME.
func clusterNameConfigMapName(nt *nomostest.NT) string {
	var configMapName string
	if nt.MultiRepo {
		// The value is defined in manifests/templates/reconciler-manager.yaml
		configMapName = "reconciler-manager"
	} else {
		// The value is defined in manifests/templates/git-importer.yaml
		return "cluster-name"
	}

	if err := nt.Validate(configMapName, configmanagement.ControllerNamespace,
		&corev1.ConfigMap{}, configMapHasClusterName(prodClusterName)); err != nil {
		nt.T.Fatal(err)
	}
	return configMapName
}

// configMapHasClusterName validates if the config map has the expected cluster name in `.data.CLUSTER_NAME`.
func configMapHasClusterName(clusterName string) nomostest.Predicate {
	return func(o core.Object) error {
		cm, ok := o.(*corev1.ConfigMap)
		if !ok {
			return nomostest.WrongTypeErr(cm, &corev1.ConfigMap{})
		}
		actual := cm.Data["CLUSTER_NAME"]
		if clusterName != actual {
			return fmt.Errorf("cluster name %q is not equal to the expected %q", actual, clusterName)
		}
		return nil
	}
}

// resourceQuotaHasHardPods validates if the resource quota has the expected hard pods in `.spec.hard.pods`.
func resourceQuotaHasHardPods(pods string) nomostest.Predicate {
	return func(o core.Object) error {
		rq, ok := o.(*corev1.ResourceQuota)
		if !ok {
			return nomostest.WrongTypeErr(rq, &corev1.ResourceQuota{})
		}
		actual := rq.Spec.Hard.Pods().String()
		if pods != actual {
			return fmt.Errorf("resource pods quota %q is not equal to the expected %q", actual, pods)
		}
		return nil
	}
}
