package e2e

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var crontabGVK = schema.GroupVersionKind{
	Group:   "stable.example.com",
	Kind:    "CronTab",
	Version: "v1",
}

func crontabCR(namespace, name string) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(crontabGVK)
	u.SetName(name)
	u.SetNamespace(namespace)
	err := unstructured.SetNestedField(u.Object, "* * * * */5", "spec", "cronSpec")
	return u, err
}

// TestStressCRD tests Config Sync can sync one CRD and 1000 namespaces successfully.
// Every namespace includes a ConfigMap and a CR.
func TestStressCRD(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured, ntopts.SkipMonoRepo, ntopts.StressTest)
	nt.T.Log("Stop the CS webhook by removing the webhook configuration")
	nomostest.StopWebhook(nt)
	nt.WaitForRepoSyncs()

	crdName := "crontabs.stable.example.com"
	nt.T.Logf("Delete the %q CRD if needed", crdName)
	nt.MustKubectl("delete", "crd", crdName, "--ignore-not-found")

	crdContent, err := ioutil.ReadFile("../testdata/customresources/changed_schema_crds/old_schema_crd.yaml")
	if err != nil {
		nt.T.Fatal(err)
	}
	nt.Root.AddFile("acme/crontab-crd.yaml", crdContent)

	labelKey := "StressTestName"
	labelValue := "TestStressCRD"
	for i := 1; i <= 1000; i++ {
		nt.Root.Add(fmt.Sprintf("acme/ns-%d.yaml", i), fake.NamespaceObject(fmt.Sprintf("foo%d", i)))
		nt.Root.Add(fmt.Sprintf("acme/cm-%d.yaml", i), fake.ConfigMapObject(
			core.Name("cm1"), core.Namespace(fmt.Sprintf("foo%d", i)), core.Label(labelKey, labelValue)))

		cr, err := crontabCR(fmt.Sprintf("foo%d", i), "cr1")
		if err != nil {
			nt.T.Fatal(err)
		}
		nt.Root.Add(fmt.Sprintf("acme/crontab-cr-%d.yaml", i), cr)
	}
	nt.Root.CommitAndPush("Add configs (one CRD and 1000 Namespaces (every namespace has one ConfigMap and one CR)")
	nt.WaitForRepoSyncs(nomostest.WithTimeout(10 * time.Minute))

	nt.T.Logf("Verify that the CronTab CRD is installed on the cluster")
	if err := nt.Validate(crdName, "", fake.CustomResourceDefinitionV1Object()); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Verify that there are exactly 1000 ConfigMaps managed by Config Sync on the cluster")
	cmList := &corev1.ConfigMapList{}

	if err := nt.Client.List(nt.Context, cmList, client.MatchingLabels{metadata.ManagedByKey: metadata.ManagedByValue, labelKey: labelValue}); err != nil {
		nt.T.Error(err)
	}
	if len(cmList.Items) != 1000 {
		nt.T.Errorf("The cluster should include 1000 ConfigMaps managed by Config Sync and having the `%s: %s` label exactly, found %v instead", labelKey, labelValue, len(cmList.Items))
	}

	nt.T.Logf("Verify that there are exactly 1000 CronTab CRs managed by Config Sync on the cluster")
	crList := &unstructured.UnstructuredList{}
	crList.SetGroupVersionKind(crontabGVK)
	if err := nt.Client.List(nt.Context, crList, client.MatchingLabels{metadata.ManagedByKey: metadata.ManagedByValue}); err != nil {
		nt.T.Error(err)
	}
	if len(crList.Items) != 1000 {
		nt.T.Errorf("The cluster should include 1000 ConfigMaps managed by Config Sync exactly, found %v instead", len(crList.Items))
	}
}

// TestStressLargeNamespace tests that Config Sync can sync a namespace including 5000 resources successfully.
func TestStressLargeNamespace(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured, ntopts.SkipMonoRepo, ntopts.StressTest)
	nt.T.Log("Stop the CS webhook by removing the webhook configuration")
	nomostest.StopWebhook(nt)

	nt.T.Log("Override the memory limit of the reconciler container of root-reconciler to 800MiB")
	rootSync := fake.RootSyncObjectV1Beta1()
	nt.MustMergePatch(rootSync, `{"spec": {"override": {"resources": [{"containerName": "reconciler", "memoryLimit": "800Mi"}]}}}`)
	nt.WaitForRepoSyncs()

	ns := "my-ns-1"
	nt.Root.Add("acme/ns.yaml", fake.NamespaceObject(ns))

	labelKey := "StressTestName"
	labelValue := "TestStressLargeNamespace"
	for i := 1; i <= 5000; i++ {
		nt.Root.Add(fmt.Sprintf("acme/cm-%d.yaml", i), fake.ConfigMapObject(
			core.Name(fmt.Sprintf("cm-%d", i)), core.Namespace(ns), core.Label(labelKey, labelValue)))
	}
	nt.Root.CommitAndPush("Add configs (5000 ConfigMaps and 1 Namespace")
	nt.WaitForRepoSyncs(nomostest.WithTimeout(10 * time.Minute))

	nt.T.Log("Verify there are 5000 ConfigMaps in the namespace")
	cmList := &corev1.ConfigMapList{}

	if err := nt.Client.List(nt.Context, cmList, &client.ListOptions{Namespace: ns}, client.MatchingLabels{labelKey: labelValue}); err != nil {
		nt.T.Error(err)
	}
	if len(cmList.Items) != 5000 {
		nt.T.Errorf("The %s namespace should include 5000 ConfigMaps having the `%s: %s` label exactly, found %v instead", ns, labelKey, labelValue, len(cmList.Items))
	}
}

// TestStressFrequentGitCommits adds 100 Git commits, and verifies that Config Sync can sync the changes in these commits successfully.
func TestStressFrequentGitCommits(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured, ntopts.StressTest)
	if nt.MultiRepo {
		nt.T.Log("Stop the CS webhook by removing the webhook configuration")
		nomostest.StopWebhook(nt)
	}

	ns := "bookstore"
	namespace := fake.NamespaceObject(ns)
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.CommitAndPush(fmt.Sprintf("add a namespace: %s", ns))
	nt.WaitForRepoSyncs()

	nt.T.Logf("Add 100 commits (every commit adds a new ConfigMap object into the %s namespace)", ns)
	labelKey := "StressTestName"
	labelValue := "TestStressFrequentGitCommits"
	for i := 0; i < 100; i++ {
		cmName := fmt.Sprintf("cm-%v", i)
		nt.Root.Add(fmt.Sprintf("acme/%s.yaml", cmName), fake.ConfigMapObject(core.Name(cmName), core.Namespace(ns), core.Label(labelKey, labelValue)))
		nt.Root.CommitAndPush(fmt.Sprintf("add %s", cmName))
	}
	nt.WaitForRepoSyncs(nomostest.WithTimeout(10 * time.Minute))

	nt.T.Logf("Verify that there are exactly 100 ConfigMaps under the %s namespace", ns)
	cmList := &corev1.ConfigMapList{}
	if err := nt.Client.List(nt.Context, cmList, &client.ListOptions{Namespace: ns}, client.MatchingLabels{labelKey: labelValue}); err != nil {
		nt.T.Error(err)
	}
	if len(cmList.Items) != 100 {
		nt.T.Errorf("The %s namespace should include 100 ConfigMaps having the `%s: %s` label exactly, found %v instead", ns, labelKey, labelValue, len(cmList.Items))
	}
}

func TestStressLargeRequest(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured, ntopts.StressTest, ntopts.SkipMonoRepo)

	crdName := "crontabs.stable.example.com"

	oldCrdFilePath := "../testdata/customresources/changed_schema_crds/old_schema_crd.yaml"
	nt.T.Logf("Apply the old CronTab CRD defined in %s", oldCrdFilePath)
	nt.MustKubectl("apply", "-f", oldCrdFilePath)

	nt.T.Logf("Wait until the old CRD is established")
	_, err := nomostest.Retry(nt.DefaultWaitTimeout, func() error {
		return nt.Validate(crdName, "", fake.CustomResourceDefinitionV1Object(), nomostest.IsEstablished)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	rootSyncFilePath := "../testdata/root-sync-crontab-crs.yaml"
	nt.T.Logf("Apply the RootSync object defined in %s", rootSyncFilePath)
	nt.MustKubectl("apply", "-f", rootSyncFilePath)

	nt.T.Logf("Verify that the source errors are truncated")
	_, err = nomostest.Retry(5*time.Minute, func() error {
		return nt.Validate("root-sync", configmanagement.ControllerNamespace, fake.RootSyncObjectV1Beta1(), truncateSourceErrors())
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	newCrdFilePath := "../testdata/customresources/changed_schema_crds/new_schema_crd.yaml"
	nt.T.Logf("Apply the new CronTab CRD defined in %s", newCrdFilePath)
	nt.MustKubectl("apply", "-f", newCrdFilePath)

	nt.T.Logf("Wait until the new CRD is established")
	_, err = nomostest.Retry(nt.DefaultWaitTimeout, func() error {
		return nt.Validate(crdName, "", fake.CustomResourceDefinitionV1Object(), nomostest.IsEstablished)
	})
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Wait for the sync to complete")
	sha1Fn := func(nt *nomostest.NT) (string, error) {
		rs := &v1alpha1.RootSync{}
		if err = nt.Get("root-sync", configmanagement.ControllerNamespace, rs); err != nil {
			return "", err
		}
		return rs.Status.Source.Commit, nil
	}
	nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(sha1Fn), nomostest.WithSyncDirectory("configs"), nomostest.WithTimeout(30*time.Minute))
}

func truncateSourceErrors() nomostest.Predicate {
	return func(o client.Object) error {
		rs, ok := o.(*v1beta1.RootSync)
		if !ok {
			return nomostest.WrongTypeErr(o, &v1beta1.RepoSync{})
		}
		for _, cond := range rs.Status.Conditions {
			if cond.Type == v1beta1.RootSyncSyncing && cond.Status == metav1.ConditionFalse && cond.Reason == "Source" &&
				reflect.DeepEqual(cond.ErrorSourceRefs, []v1beta1.ErrorSource{v1beta1.SourceError}) && cond.ErrorSummary.Truncated {
				return nil
			}
		}
		return errors.Errorf("the source errors should be truncated")
	}
}
