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

package e2e

import (
	"fmt"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"kpt.dev/configsync/e2e/nomostest"
	"kpt.dev/configsync/e2e/nomostest/ntopts"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/importer/analyzer/validation/nonhierarchical"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/status"
	"kpt.dev/configsync/pkg/testing/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testLabels = client.MatchingLabels{"test-case": "hydration"}
var expectedBuiltinOrigin = "configuredIn: kustomization.yaml\nconfiguredBy:\n  apiVersion: builtin\n  kind: HelmChartInflationGenerator\n"
var expectedKrmFnOrigin = "configuredIn: kustomization.yaml\nconfiguredBy:\n  apiVersion: fn.kpt.dev/v1alpha1\n  kind: RenderHelmChart\n  name: demo\n"

func TestHydrateKustomizeComponents(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the kustomize components root directory")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/kustomize-components", ".")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the kustomize-components directory")
	rs := fake.RootSyncObjectV1Beta1(configsync.RootSyncName)
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "kustomize-components"}}}`)

	nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "kustomize-components"}))

	nt.T.Log("Validate resources are synced")
	var expectedNamespaces = []string{"tenant-a", "tenant-b", "tenant-c"}
	validateNamespaces(nt, expectedNamespaces, "path: base/namespace.yaml\n")
	for _, ns := range expectedNamespaces {
		if err := nt.Validate("deny-all", ns, &networkingv1.NetworkPolicy{}, nomostest.HasAnnotation(metadata.KustomizeOrigin, "path: base/networkpolicy.yaml\n")); err != nil {
			nt.T.Error(err)
		}
		if err := nt.Validate("tenant-admin", ns, &rbacv1.Role{}, nomostest.HasAnnotation(metadata.KustomizeOrigin, "path: base/role.yaml\n")); err != nil {
			nt.T.Error(err)
		}
		if err := nt.Validate("tenant-admin-rolebinding", ns, &rbacv1.RoleBinding{}, nomostest.HasAnnotation(metadata.KustomizeOrigin, "path: base/rolebinding.yaml\n")); err != nil {
			nt.T.Error(err)
		}
	}

	nt.T.Log("Remove kustomization.yaml to make the sync fail")
	nt.RootRepos[configsync.RootSyncName].Remove("./kustomize-components/kustomization.yml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("remove the Kustomize configuration to make the sync fail")
	nt.WaitForRootSyncRenderingError(configsync.RootSyncName, status.ActionableHydrationErrorCode, "")

	nt.T.Log("Add kustomization.yaml back")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/kustomize-components/kustomization.yml", "./kustomize-components/kustomization.yml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add kustomization.yml back")
	nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "kustomize-components"}))

	nt.T.Log("Make kustomization.yaml invalid")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/invalid-kustomization.yaml", "./kustomize-components/kustomization.yml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("update kustomization.yaml to make it invalid")
	nt.WaitForRootSyncRenderingError(configsync.RootSyncName, status.ActionableHydrationErrorCode, "")
}

func TestHydrateHelmComponents(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the helm components root directory")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/helm-components", ".")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the helm-components directory")
	rs := fake.RootSyncObjectV1Beta1(configsync.RootSyncName)
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "helm-components"}}}`)

	nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "helm-components"}))

	nt.T.Log("Validate resources are synced")
	if err := nt.Validate("my-coredns-coredns", "coredns", &appsv1.Deployment{}, containerImagePullPolicy("IfNotPresent"), nomostest.HasAnnotation(metadata.KustomizeOrigin, expectedBuiltinOrigin)); err != nil {
		nt.T.Fatal(err)
	}
	if err := nt.Validate("my-ingress-nginx-controller", "ingress-nginx", &appsv1.Deployment{}, containerImagePullPolicy("IfNotPresent"), nomostest.HasAnnotation(metadata.KustomizeOrigin, expectedBuiltinOrigin)); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Use a remote values.yaml file from a public repo")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/helm-components-remote-values-kustomization.yaml", "./helm-components/kustomization.yaml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Render with a remote values.yaml file from a public repo")
	nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "helm-components"}))
	if err := nt.Validate("my-coredns-coredns", "coredns", &appsv1.Deployment{}, containerImagePullPolicy("Always"), nomostest.HasAnnotation(metadata.KustomizeOrigin, expectedBuiltinOrigin)); err != nil {
		nt.T.Fatal(err)
	}

	// TODO(b/209458334) Skip the following test when running on GKE Autopilot clusters.
	if !nt.IsGKEAutopilot {
		nt.T.Log("Use the render-helm-chart function to render the charts")
		nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/krm-function-helm-components-kustomization.yaml", "./helm-components/kustomization.yaml")
		nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update kustomization.yaml to use the render-helm-chart function")
		nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "helm-components"}))
		if err := nt.Validate("my-coredns-coredns", "coredns", &appsv1.Deployment{}, containerImagePullPolicy("IfNotPresent"), nomostest.HasAnnotation(metadata.KustomizeOrigin, expectedKrmFnOrigin)); err != nil {
			nt.T.Fatal(err)
		}

		nt.T.Log("Use the render-helm-chart function to render the charts with a remote values.yaml file")
		nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/krm-function-helm-components-remote-values-kustomization.yaml", "./helm-components/kustomization.yaml")
		nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update kustomization.yaml to use the render-helm-chart function with a remote values.yaml file from a public repo")
		nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "helm-components"}))
		if err := nt.Validate("my-coredns-coredns", "coredns", &appsv1.Deployment{}, containerImagePullPolicy("Always"), nomostest.HasAnnotation(metadata.KustomizeOrigin, expectedKrmFnOrigin)); err != nil {
			nt.T.Fatal(err)
		}
	}
}

func TestHydrateHelmOverlay(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the helm-overlay root directory")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/helm-overlay", ".")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the helm-overlay directory")
	rs := fake.RootSyncObjectV1Beta1(configsync.RootSyncName)
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "helm-overlay"}}}`)

	nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "helm-overlay"}))

	nt.T.Log("Validate resources are synced")
	if err := nt.Validate("my-coredns-coredns", "coredns", &appsv1.Deployment{},
		nomostest.HasAnnotation("hydration-tool", "kustomize"),
		nomostest.HasLabel("team", "coredns"),
		nomostest.HasAnnotation("client.lifecycle.config.k8s.io/mutation", "ignore"),
		nomostest.HasAnnotation(metadata.KustomizeOrigin, "configuredIn: base/kustomization.yaml\nconfiguredBy:\n  apiVersion: builtin\n  kind: HelmChartInflationGenerator\n"),
		nomostest.HasLabel("test-case", "hydration")); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Make the hydration fail by checking in an invalid kustomization.yaml")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/resource-duplicate/kustomization.yaml", "./helm-overlay/kustomization.yaml")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/resource-duplicate/namespace_tenant-a.yaml", "./helm-overlay/namespace_tenant-a.yaml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update kustomization.yaml with duplicated resources")
	nt.WaitForRootSyncRenderingError(configsync.RootSyncName, status.ActionableHydrationErrorCode, "")

	nt.T.Log("Make the parsing fail by checking in a deprecated group and kind")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/deprecated-GK/kustomization.yaml", "./helm-overlay/kustomization.yaml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update kustomization.yaml to render a deprecated group and kind")
	nt.WaitForRootSyncSourceError(configsync.RootSyncName, nonhierarchical.DeprecatedGroupKindErrorCode, "")

	// TODO(b/209458334) Skip the following test when running on GKE Autopilot clusters.
	if !nt.IsGKEAutopilot {
		nt.T.Log("Use the render-helm-chart function to render the charts")
		nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/helm-overlay/kustomization.yaml", "./helm-overlay/kustomization.yaml")
		nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/krm-function-helm-overlay-kustomization.yaml", "./helm-overlay/base/kustomization.yaml")
		nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update kustomization.yaml to use the render-helm-chart function")
		nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "helm-overlay"}))

		nt.T.Log("Make the parsing fail again by checking in a deprecated group and kind with the render-helm-chart function")
		nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/krm-function-deprecated-GK-kustomization.yaml", "./helm-overlay/kustomization.yaml")
		nt.RootRepos[configsync.RootSyncName].CommitAndPush("Update kustomization.yaml to render a deprecated group and kind with the render-helm-chart function")
		nt.WaitForRootSyncSourceError(configsync.RootSyncName, nonhierarchical.DeprecatedGroupKindErrorCode, "")
	}
}

func TestHydrateResourcesInRelativePath(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the root directory")
	nt.RootRepos[configsync.RootSyncName].Copy("../testdata/hydration/relative-path", ".")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the relative-path directory")
	rs := fake.RootSyncObjectV1Beta1(configsync.RootSyncName)
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "relative-path/overlays/dev"}}}`)

	nt.WaitForRepoSyncs(nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: "relative-path/overlays/dev"}))

	nt.T.Log("Validating resources are synced")
	if err := nt.Validate("foo", "", &corev1.Namespace{}, nomostest.HasAnnotation(metadata.KustomizeOrigin, "path: ../../base/foo/namespace.yaml\n")); err != nil {
		nt.T.Error(err)
	}
	if err := nt.Validate("pod-creators", "foo", &rbacv1.RoleBinding{}, nomostest.HasAnnotation(metadata.KustomizeOrigin, "path: ../../base/foo/pod-creator-rolebinding.yaml\n")); err != nil {
		nt.T.Error(err)
	}
	if err := nt.Validate("foo-ksa-dev", "foo", &corev1.ServiceAccount{}, nomostest.HasAnnotation(metadata.KustomizeOrigin, "path: ../../base/foo/serviceaccount.yaml\n")); err != nil {
		nt.T.Error(err)
	}
	if err := nt.Validate("pod-creator", "", &rbacv1.ClusterRole{}, nomostest.HasAnnotation(metadata.KustomizeOrigin, "path: ../../base/pod-creator-clusterrole.yaml\n")); err != nil {
		nt.T.Error(err)
	}
}

func validateNamespaces(nt *nomostest.NT, expectedNamespaces []string, expectedOrigin string) {
	namespaces := &corev1.NamespaceList{}
	if err := nt.List(namespaces, testLabels); err != nil {
		nt.T.Error(err)
	}
	var actualNamespaces []string
	for _, ns := range namespaces.Items {
		if ns.Status.Phase == corev1.NamespaceActive {
			actualNamespaces = append(actualNamespaces, ns.Name)
		}
		origin, ok := ns.Annotations[metadata.KustomizeOrigin]
		if !ok {
			nt.T.Errorf("expected annotation[%q], but not found", metadata.KustomizeOrigin)
		}
		if origin != expectedOrigin {
			nt.T.Errorf("expected annotation[%q] to be %q, but got '%s'", metadata.KustomizeOrigin, expectedOrigin, origin)
		}
	}
	if !reflect.DeepEqual(actualNamespaces, expectedNamespaces) {
		nt.T.Errorf("expected namespaces: %v, but got: %v", expectedNamespaces, actualNamespaces)
	}
}

func containerImagePullPolicy(policy string) nomostest.Predicate {
	return func(o client.Object) error {
		rq, ok := o.(*appsv1.Deployment)
		if !ok {
			return nomostest.WrongTypeErr(rq, &appsv1.Deployment{})
		}

		actual := rq.Spec.Template.Spec.Containers[0].ImagePullPolicy
		if policy != string(actual) {
			return fmt.Errorf("container policy %q is not equal to the expected %q", actual, policy)
		}
		return nil
	}
}
