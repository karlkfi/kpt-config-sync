package e2e

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testLabels = client.MatchingLabels{"test-case": "hydration"}

func TestHydrateKustomizeComponents(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the kustomize components root directory")
	nt.Root.Copy("../testdata/hydration/kustomize-components", ".")
	nt.Root.CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the kustomize-components branch")
	rs := fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "kustomize-components"}}}`)

	nt.WaitForRepoSyncs()

	nt.T.Log("Validate resources are synced")
	var expectedNamespaces = []string{"tenant-a", "tenant-b", "tenant-c"}
	validateNamespaces(nt, expectedNamespaces)
	for _, ns := range expectedNamespaces {
		if err := nt.Validate("deny-all", ns, &networkingv1.NetworkPolicy{}); err != nil {
			nt.T.Error(err)
		}
		if err := nt.Validate("tenant-admin", ns, &rbacv1.Role{}); err != nil {
			nt.T.Error(err)
		}
		if err := nt.Validate("tenant-admin-rolebinding", ns, &rbacv1.RoleBinding{}); err != nil {
			nt.T.Error(err)
		}
	}

	nt.T.Log("Remove kustomization.yaml to make the sync fail")
	nt.Root.Remove("./kustomize-components/kustomization.yml")
	nt.Root.CommitAndPush("remove the Kustomize configuration to make the sync fail")

	nt.WaitForRootSyncSourceError(status.PathErrorCode)

	nt.T.Log("Add kustomization.yaml back")
	nt.Root.Copy("../testdata/hydration/kustomize-components/kustomization.yml", "./kustomize-components/kustomization.yml")
	nt.Root.CommitAndPush("add kustomization.yml back")

	nt.WaitForRepoSyncs()
}

func TestHydrateHelmComponents(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the helm components root directory")
	nt.Root.Copy("../testdata/hydration/helm-components", ".")
	nt.Root.CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the helm-components branch")
	rs := fake.RootSyncObjectV1Beta1()
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "helm-components"}}}`)

	nt.WaitForRepoSyncs()

	nt.T.Log("Validate resources are synced")
	if err := nt.Validate("my-prometheus-alertmanager", "monitoring", &appsv1.Deployment{}, containerImagePullPolicy("IfNotPresent")); err != nil {
		nt.T.Fatal(err)
	}
	if err := nt.Validate("my-cert-manager", "cert-manager", &appsv1.Deployment{}, containerImagePullPolicy("IfNotPresent")); err != nil {
		nt.T.Fatal(err)
	}

	nt.T.Log("Use a remote values.yaml file from a public repo")
	nt.Root.Copy("../testdata/hydration/helm-components-remote-values-kustomization.yaml", "./helm-components/kustomization.yaml")
	nt.Root.CommitAndPush("Render with a remote values.yaml file from a public repo")
	nt.WaitForRepoSyncs()
	if err := nt.Validate("my-cert-manager", "cert-manager", &appsv1.Deployment{}, containerImagePullPolicy("Always")); err != nil {
		nt.T.Fatal(err)
	}

	// TODO: use a remote values.yaml file from a private repo
}

func TestHydrateHelmOverlay(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the helm-overlay root directory")
	nt.Root.Copy("../testdata/hydration/helm-overlay", ".")
	nt.Root.CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the helm-overlay branch")
	rs := fake.RootSyncObject()
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "helm-overlay"}}}`)

	nt.WaitForRepoSyncs()

	nt.T.Log("Validate resources are synced")
	if err := nt.Validate("my-prometheus-alertmanager", "monitoring", &appsv1.Deployment{},
		nomostest.HasAnnotation("hydration-tool", "kustomize"),
		nomostest.HasLabel("team", "monitoring"),
		nomostest.HasAnnotation("client.lifecycle.config.k8s.io/mutation", "ignore"),
		nomostest.HasLabel("test-case", "hydration")); err != nil {
		nt.T.Fatal(err)
	}
}

func TestHydrateRemoteResources(t *testing.T) {
	nt := nomostest.New(t,
		ntopts.SkipMonoRepo,
		ntopts.Unstructured,
	)

	nt.T.Log("Add the remote-base root directory")
	nt.Root.Copy("../testdata/hydration/remote-base", ".")
	nt.Root.CommitAndPush("add DRY configs to the repository")

	nt.T.Log("Update RootSync to sync from the remote-base branch")
	rs := fake.RootSyncObjectV1Beta1()
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "remote-base"}}}`)

	nt.WaitForRepoSyncs()

	nt.T.Log("Validate resources are synced")
	var expectedNamespaces = []string{"tenant-a"}
	validateNamespaces(nt, expectedNamespaces)

	nt.T.Log("Update kustomization.yaml to use a remote overlay")
	nt.Root.Copy("../testdata/hydration/remote-overlay-kustomization.yaml", "./remote-base/kustomization.yaml")
	nt.Root.CommitAndPush("Update kustomization.yaml to use a remote overlay")
	nt.WaitForRepoSyncs()

	nt.T.Log("Validate resources are synced")
	expectedNamespaces = []string{"tenant-b"}
	validateNamespaces(nt, expectedNamespaces)

	// Update kustomization.yaml to use remote resources
	nt.Root.Copy("../testdata/hydration/remote-resources-kustomization.yaml", "./remote-base/kustomization.yaml")
	nt.Root.CommitAndPush("Update kustomization.yaml to use remote resources")
	nt.WaitForRepoSyncs()

	nt.T.Log("Validate resources are synced")
	expectedNamespaces = []string{"tenant-a", "tenant-b", "tenant-c"}
	validateNamespaces(nt, expectedNamespaces)
}

func validateNamespaces(nt *nomostest.NT, expectedNamespaces []string) {
	namespaces := &corev1.NamespaceList{}
	if err := nt.List(namespaces, testLabels); err != nil {
		nt.T.Error(err)
	}
	var actualNamespaces []string
	for _, ns := range namespaces.Items {
		if ns.Status.Phase == corev1.NamespaceActive {
			actualNamespaces = append(actualNamespaces, ns.Name)
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
