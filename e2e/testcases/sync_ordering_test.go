package e2e

import (
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cli-utils/pkg/object/dependson"
)

// This file includes e2e tests for sync ordering (design doc: go/cs-sync-ordering).
// The sync ordering feature is only supported in the multi-repo mode.

func TestSyncOrdering(t *testing.T) {
	nt := nomostest.New(t, ntopts.Unstructured, ntopts.SkipMonoRepo)

	namespaceName := "bookstore"
	nt.T.Logf("Remove the namespace %q if it already exists", namespaceName)
	nt.MustKubectl("delete", "ns", namespaceName, "--ignore-not-found")

	nt.T.Log("A new test: verify that an object is created after its dependency (cm1 and cm2 both are in the Git repo, but don't exist on the cluster. cm2 depends on cm1.)")
	nt.T.Logf("Add the namespace, cm1, and cm2 (cm2 depends on cm1)")
	namespace := fake.NamespaceObject(namespaceName)
	cm1Name := "cm1"
	cm2Name := "cm2"
	nt.Root.Add("acme/ns.yaml", namespace)
	nt.Root.Add("acme/cm1.yaml", fake.ConfigMapObject(core.Name(cm1Name), core.Namespace(namespaceName)))
	// cm2 depends on cm1
	nt.Root.Add("acme/cm2.yaml", fake.ConfigMapObject(core.Name(cm2Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm1")))
	nt.Root.CommitAndPush("Add the namespace, cm1, and cm2 (cm2 depends on cm1)")
	nt.WaitForRepoSyncs()

	ns := &corev1.Namespace{}
	if err := nt.Get(namespaceName, "", ns); err != nil {
		nt.T.Fatal(err)
	}

	cm1 := &corev1.ConfigMap{}
	if err := nt.Get(cm1Name, namespaceName, cm1); err != nil {
		nt.T.Fatal(err)
	}

	cm2 := &corev1.ConfigMap{}
	if err := nt.Get(cm2Name, namespaceName, cm2); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Verify that the namespace is created before the configmaps in it")
	if cm1.CreationTimestamp.Before(&ns.CreationTimestamp) {
		nt.T.Fatalf("a namespace (%s) should be created before a ConfigMap (%s) in it", core.GKNN(ns), core.GKNN(cm1))
	}

	if cm2.CreationTimestamp.Before(&ns.CreationTimestamp) {
		nt.T.Fatalf("a namespace (%s) should be created before a ConfigMap (%s) in it", core.GKNN(ns), core.GKNN(cm2))
	}

	nt.T.Logf("Verify that cm1 is created before cm2")
	if cm2.CreationTimestamp.Before(&cm1.CreationTimestamp) {
		nt.T.Fatalf("an object (%s) should be created after its dependency (%s)", core.GKNN(cm2), core.GKNN(cm1))
	}

	nt.T.Logf("Verify that cm2 has the dependsOn annotation")
	if err := nt.Validate(cm2Name, namespaceName, &corev1.ConfigMap{}, nomostest.HasAnnotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm1")); err != nil {
		nt.T.Fatal(err)
	}

	// There are 2 configmaps in the namespace at this point: cm1, cm2.
	// The dependency graph is:
	//   * cm2 depends on cm1

	nt.T.Log("A new test: verify that an object can declare dependency on an existing object (cm1 and cm3 both are in the Git repo, cm3 depends on cm1. cm1 already exists on the cluster, cm3 does not.)")
	nt.T.Log("Add cm3, which depends on an existing object, cm1")
	// cm3 depends on cm1
	cm3Name := "cm3"
	nt.Root.Add("acme/cm3.yaml", fake.ConfigMapObject(core.Name(cm3Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm1")))
	nt.Root.CommitAndPush("add cm3, which depends on an existing object, cm1")
	nt.WaitForRepoSyncs()

	nt.T.Logf("Verify that cm1 is created before cm3")
	cm1 = &corev1.ConfigMap{}
	if err := nt.Get(cm1Name, namespaceName, cm1); err != nil {
		nt.T.Fatal(err)
	}

	cm3 := &corev1.ConfigMap{}
	if err := nt.Get(cm3Name, namespaceName, cm3); err != nil {
		nt.T.Fatal(err)
	}
	if cm3.CreationTimestamp.Before(&cm1.CreationTimestamp) {
		nt.T.Fatalf("an object (%s) should be created after its dependency (%s)", core.GKNN(cm3), core.GKNN(cm1))
	}

	// There are 3 configmaps in the namespace at this point: cm1, cm2, cm3
	// The dependency graph is:
	//   * cm2 depends on cm1
	//   * cm3 depends on cm1

	nt.T.Log("A new test: verify that an existing object can declare dependency on a non-existing object (cm1 and cm0 both are in the Git repo, cm1 depends on cm0. cm1 already exists on the cluster, cm0 does not.)")
	nt.T.Log("add a new configmap, cm0; and add the dependsOn annotation to cm1")
	// cm1 depends on cm0
	cm0Name := "cm0"
	nt.Root.Add("acme/cm0.yaml", fake.ConfigMapObject(core.Name(cm0Name), core.Namespace(namespaceName)))
	nt.Root.Add("acme/cm1.yaml", fake.ConfigMapObject(core.Name(cm1Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0")))
	nt.Root.CommitAndPush("add a new configmap, cm0; and add the dependsOn annotation to cm1")
	nt.WaitForRepoSyncs()

	nt.T.Log("Verify that cm1 is created before cm0")
	cm1 = &corev1.ConfigMap{}
	if err := nt.Get(cm1Name, namespaceName, cm1); err != nil {
		nt.T.Fatal(err)
	}

	cm0 := &corev1.ConfigMap{}
	if err := nt.Get(cm3Name, namespaceName, cm0); err != nil {
		nt.T.Fatal(err)
	}
	if cm0.CreationTimestamp.Before(&cm1.CreationTimestamp) {
		nt.T.Fatalf("Declaring the dependency of an existing object (%s) on a non-existing object (%s) should not cause the existing object to be recreated", core.GKNN(cm1), core.GKNN(cm0))
	}

	// There are 4 configmaps in the namespace at this point: cm0, cm1, cm2, cm3
	// The dependency graph is:
	//   * cm1 depends on cm0
	//   * cm2 depends on cm1
	//   * cm3 depends on cm1

	nt.T.Log("A new test: verify that Config Sync reports an error when a cyclic dependency is encountered (a cyclic dependency between cm0, cm1, and cm2. cm1 depends on cm0; cm2 depends on cm1; cm0 depends on cm2)")
	nt.T.Log("Create a cyclic dependency between cm0, cm1, and cm2")
	nt.Root.Add("acme/cm0.yaml", fake.ConfigMapObject(core.Name(cm0Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm2")))
	nt.Root.CommitAndPush("Create a cyclic dependency between cm0, cm1, and cm2")
	nt.WaitForRootSyncSyncError(applier.ApplierErrorCode, "cyclic dependency")

	nt.T.Log("Verify that cm0 does not have the dependsOn annotation")
	if err := nt.Validate(cm0Name, namespaceName, &corev1.ConfigMap{}, nomostest.MissingAnnotation(dependson.Annotation)); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Remove the cyclic dependency from the Git repo")
	nt.Root.Add("acme/cm0.yaml", fake.ConfigMapObject(core.Name(cm0Name), core.Namespace(namespaceName)))
	nt.Root.CommitAndPush("Remove the cyclic dependency from the Git repo")
	nt.WaitForRepoSyncs()

	// There are 4 configmaps in the namespace at this point: cm0, cm1, cm2, cm3.
	// The dependency graph is:
	//   * cm1 depends on cm0
	//   * cm2 depends on cm1
	//   * cm3 depends on cm1

	nt.T.Log("A new test: verify that an object can be removed without affecting its dependency (cm3 depends on cm1, and both cm3 and cm1 exist in the Git repo and on the cluster.)")
	nt.T.Log("Remove cm3")
	nt.Root.Remove("acme/cm3.yaml")
	nt.Root.CommitAndPush("Remove cm3")
	nt.WaitForRepoSyncs()

	nt.T.Log("Verify that cm3 is removed")
	if _, err := nomostest.Retry(nt.DefaultWaitTimeout, func() error {
		return nt.ValidateNotFound(cm3Name, namespaceName, &corev1.ConfigMap{})
	}); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Verify that cm1 is still on the cluster")
	if err := nt.Validate(cm1Name, namespaceName, &corev1.ConfigMap{}); err != nil {
		nt.T.Fatal(err)
	}

	// There are 3 configmaps in the namespace at this point: cm0, cm1, cm2.
	// The dependency graph is:
	//   * cm1 depends on cm0
	//   * cm2 depends on cm1

	nt.T.Log("A new test: verify that an object and its dependency can be removed together (cm1 and cm2 both exist in the Git repo and on the cluster. cm2 depends on cm1.)")
	nt.T.Log("Remove cm1 and cm2")
	nt.Root.Remove("acme/cm1.yaml")
	nt.Root.Remove("acme/cm2.yaml")
	nt.Root.CommitAndPush("Remove cm1 and cm2")
	nt.WaitForRepoSyncs()

	nt.T.Log("Verify that cm1 is removed")
	if _, err := nomostest.Retry(nt.DefaultWaitTimeout, func() error {
		return nt.ValidateNotFound(cm1Name, namespaceName, &corev1.ConfigMap{})
	}); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Verify that cm2 is removed")
	if _, err := nomostest.Retry(nt.DefaultWaitTimeout, func() error {
		return nt.ValidateNotFound(cm2Name, namespaceName, &corev1.ConfigMap{})
	}); err != nil {
		nt.T.Fatal(err)
	}

	// There are 1 configmap in the namespace at this point: cm0.

	nt.T.Log("Add cm0, cm1, cm2, and cm3")
	nt.Root.Add("acme/cm1.yaml", fake.ConfigMapObject(core.Name(cm1Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0")))
	nt.Root.Add("acme/cm2.yaml", fake.ConfigMapObject(core.Name(cm2Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0")))
	nt.Root.Add("acme/cm3.yaml", fake.ConfigMapObject(core.Name(cm3Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0")))
	nt.Root.CommitAndPush("Add cm0, cm1, cm2, and cm3")
	nt.WaitForRepoSyncs()

	nt.T.Logf("Verify that cm1 has the dependsOn annotation, and depends on cm0")
	if err := nt.Validate(cm1Name, namespaceName, &corev1.ConfigMap{}, nomostest.HasAnnotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0")); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Verify that cm2 has the dependsOn annotation, and depends on cm0")
	if err := nt.Validate(cm2Name, namespaceName, &corev1.ConfigMap{}, nomostest.HasAnnotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0")); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Logf("Verify that cm3 has the dependsOn annotation, and depends on cm0")
	if err := nt.Validate(cm3Name, namespaceName, &corev1.ConfigMap{}, nomostest.HasAnnotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0")); err != nil {
		nt.T.Fatal(err)
	}

	// There are 4 configmaps in the namespace at this point: cm0, cm1, cm2 and cm3.
	// The dependency graph is:
	//   * cm1 depends on cm0
	//   * cm2 depends on cm0
	//   * cm3 depends on cm0

	nt.T.Log("A new test: verify that an object can be disabled without affecting its dependency")
	nt.T.Log("Disable cm3 by adding the `configmanagement.gke.io/managed: disabled` annotation")
	nt.Root.Add("acme/cm3.yaml", fake.ConfigMapObject(core.Name(cm3Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0"),
		core.Annotation(metadata.ResourceManagementKey, metadata.ResourceManagementDisabled)))
	nt.Root.CommitAndPush("Disable cm3 by adding the `configmanagement.gke.io/managed: disabled` annotation")
	nt.WaitForRepoSyncs()

	nt.T.Log("Verify that cm3 no longer has the CS metadata")
	if err := nt.Validate(cm3Name, namespaceName, &corev1.ConfigMap{}, nomostest.NoConfigSyncMetadata()); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Verify that cm0 still has the CS metadata")
	if err := nt.Validate(cm0Name, namespaceName, &corev1.ConfigMap{}, nomostest.HasAllNomosMetadata(true)); err != nil {
		nt.T.Fatal(err)
	}

	// There are 4 configmaps in the namespace at this point: cm0, cm1, cm2 and cm3.
	// The inventory tracks 3 configmaps: cm0, cm1, cm2. The dependency graph is:
	//   * cm1 depends on cm0
	//   * cm2 depends on cm0

	nt.T.Log("A new test: verify that the dependsOn annotation can be removed from an object without affecting its dependency")
	nt.T.Log("Remove the dependsOn annotation from cm2")
	nt.Root.Add("acme/cm2.yaml", fake.ConfigMapObject(core.Name(cm2Name), core.Namespace(namespaceName)))
	nt.Root.CommitAndPush("Remove the dependsOn annotation from cm2")
	nt.WaitForRepoSyncs()

	nt.T.Log("Verify that cm2 no longer has the dependsOn annotation")
	if err := nt.Validate(cm2Name, namespaceName, &corev1.ConfigMap{}, nomostest.MissingAnnotation(dependson.Annotation)); err != nil {
		nt.T.Fatal(err)
	}

	// There are 4 configmaps in the namespace at this point: cm0, cm1, cm2 and cm3.
	// The inventory tracks 3 configmaps: cm0, cm1, cm2. The dependency graph is:
	//   * cm1 depends on cm0

	nt.T.Log("A new test: verify that an object and its dependency can be disabled together")
	nt.T.Log("Disable both cm1 and cm0 by adding the `configmanagement.gke.io/managed: disabled` annotation")
	nt.Root.Add("acme/cm0.yaml", fake.ConfigMapObject(core.Name(cm0Name), core.Namespace(namespaceName),
		core.Annotation(metadata.ResourceManagementKey, metadata.ResourceManagementDisabled)))
	nt.Root.Add("acme/cm1.yaml", fake.ConfigMapObject(core.Name(cm1Name), core.Namespace(namespaceName),
		core.Annotation(dependson.Annotation, "/namespaces/bookstore/ConfigMap/cm0"),
		core.Annotation(metadata.ResourceManagementKey, metadata.ResourceManagementDisabled)))
	nt.Root.CommitAndPush("Disable both cm1 and cm0 by adding the `configmanagement.gke.io/managed: disabled` annotation")
	nt.WaitForRepoSyncs()

	nt.T.Log("Verify that cm1 no longer has the CS metadata")
	if err := nt.Validate(cm1Name, namespaceName, &corev1.ConfigMap{}, nomostest.NoConfigSyncMetadata()); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Verify that cm0 no longer has the CS metadata")
	if err := nt.Validate(cm0Name, namespaceName, &corev1.ConfigMap{}, nomostest.NoConfigSyncMetadata()); err != nil {
		nt.T.Fatal(err)
	}
}
