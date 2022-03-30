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
	"os"
	"os/exec"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"kpt.dev/configsync/e2e/nomostest"
	"kpt.dev/configsync/e2e/nomostest/ntopts"
	"kpt.dev/configsync/pkg/api/configmanagement"
	"kpt.dev/configsync/pkg/api/configsync"
	"kpt.dev/configsync/pkg/declared"
	"kpt.dev/configsync/pkg/metadata"
	"kpt.dev/configsync/pkg/reconcilermanager"
	"kpt.dev/configsync/pkg/testing/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	csrRepo    = "https://source.developers.google.com/p/stolos-dev/r/csr-auth-test"
	syncBranch = "main"
)

// TestGCENode tests the `genode` auth type.
// The test will run on a GKE cluster only with following pre-requisites:
// 1. Workload Identity is NOT enabled
// 2. Access scopes for the nodes in the cluster must include `cloud-source-repos-ro`.
// 3. The Compute Engine default service account `PROJECT_ID-compute@developer.gserviceaccount.com` has `source.reader` access to Cloud Source Repository.
// Public documentation: https://cloud.google.com/anthos-config-management/docs/how-to/installing-config-sync#git-creds-secret
func TestGCENode(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.Unstructured,
		ntopts.RequireGKE(t), ntopts.GCENodeTest)

	origRepoURL := nt.GitProvider.SyncURL(nt.RootRepos[configsync.RootSyncName].RemoteRepoName)
	tenant := "tenant-a"
	rs := fake.RootSyncObjectV1Beta1(configsync.RootSyncName)
	nt.T.Log("Update RootSync to sync from a CSR repo")
	nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "%s", "branch": "%s", "repo": "%s", "auth": "gcenode", "secretRef": {"name": ""}}, "sourceFormat": "unstructured"}}`,
		tenant, syncBranch, csrRepo))

	nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(nomostest.RemoteRepoRootSha1Fn),
		nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: tenant}))
	validateTenant(nt, string(declared.RootReconciler), tenant)
	validateFWICredentials(nt, nomostest.DefaultRootReconcilerName, fwiAnnotationAbsent)

	// Change the rs back so that it works in the shared test environment.
	nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "acme", "branch": "main", "repo": "%s", "auth": "ssh","gcpServiceAccountEmail": "", "secretRef": {"name": "git-creds"}}, "sourceFormat": "hierarchy"}}`, origRepoURL))
}

// TestWorkloadIdentity tests the `gcpserviceaccount` auth type with both GKE
// Workload Identity and Fleet Workload Identity (in-project and cross-project).
//  The test will run on a GKE cluster only with following pre-requisites
// 1. Workload Identity is enabled.
// 2. Access scopes for the nodes in the cluster must include `cloud-source-repos-ro`.
// 3. The Google service account `e2e-test-csr-reader@stolos-dev.iam.gserviceaccount.com` is created with the `roles/source.reader` role to access to CSR.
// 4. An IAM policy binding is created between the Google service account and the Kubernetes service accounts with the `roles/iam.workloadIdentityUser` role.
//   gcloud iam service-accounts add-iam-policy-binding --project=stolos-dev \
//      --role roles/iam.workloadIdentityUser \
//      --member "serviceAccount:stolos-dev.svc.id.goog[config-management-system/root-reconciler]" \
//      --member="serviceAccount:cs-dev-hub.svc.id.goog[config-management-system/root-reconciler]" \
//      e2e-test-csr-reader@stolos-dev.iam.gserviceaccount.com
// 5. The cross-project fleet host project 'cs-dev-hub' is created.
// 6. The following environment variables are set: GCP_PROJECT, GCP_CLUSTER, GCP_REGION|GCP_ZONE.
func TestWorkloadIdentity(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.Unstructured, ntopts.RequireGKE(t))

	origRepoURL := nt.GitProvider.SyncURL(nt.RootRepos[configsync.RootSyncName].RemoteRepoName)
	gcpProject := os.Getenv("GCP_PROJECT")
	if gcpProject == "" {
		t.Fatal("Environment variable 'GCP_PROJECT' is required for this test case")
	}
	gcpCluster := os.Getenv("GCP_CLUSTER")
	if gcpCluster == "" {
		t.Fatal("Environment variable 'GCP_CLUSTER' is required for this test case")
	}
	gkeRegion := os.Getenv("GCP_REGION")
	gkeZone := os.Getenv("GCP_ZONE")
	if gkeRegion == "" && gkeZone == "" {
		t.Fatal("Environment variable 'GCP_REGION' or 'GCP_ZONE' is required for this test case")
	}

	gsaEmail := "e2e-test-csr-reader@stolos-dev.iam.gserviceaccount.com"
	tenant := "tenant-a"
	rs := fake.RootSyncObjectV1Beta1(configsync.RootSyncName)
	nt.T.Logf("Update RootSync to sync %s from a CSR repo", tenant)
	nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "%s", "branch": "%s", "repo": "%s", "auth": "gcpserviceaccount", "gcpServiceAccountEmail": "%s", "secretRef": {"name": ""}}, "sourceFormat": "unstructured"}}`,
		tenant, syncBranch, csrRepo, gsaEmail))

	gkeURI := "https://container.googleapis.com/v1/projects/" + gcpProject
	fleetMembership := fmt.Sprintf("%s-%s", gcpProject, gcpCluster)

	if gkeRegion != "" {
		gkeURI += fmt.Sprintf("/locations/%s/clusters/%s", gkeRegion, gcpCluster)
	} else {
		gkeURI += fmt.Sprintf("/zones/%s/clusters/%s", gkeZone, gcpCluster)
	}
	nt.T.Log("Register the cluster to a fleet in the same project")
	out, err := exec.Command("gcloud", "beta", "container", "hub", "memberships", "register", fleetMembership, "--project", gcpProject, "--gke-uri", gkeURI, "--enable-workload-identity").CombinedOutput()
	if err != nil {
		nt.T.Fatalf("%s: %v", string(out), err)
	}
	nt.T.Log("Restart the reconciler-manager to pick up the Membership")
	nomostest.DeletePodByLabel(nt, "app", reconcilermanager.ManagerName)

	nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(nomostest.RemoteRepoRootSha1Fn),
		nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: tenant}))
	validateTenant(nt, string(declared.RootReconciler), tenant)
	validateFWICredentials(nt, nomostest.DefaultRootReconcilerName, fwiAnnotationExists)

	nt.T.Log("Unregister the cluster from the fleet in the same project")
	out, err = exec.Command("gcloud", "beta", "container", "hub", "memberships", "unregister", fleetMembership, "--project", gcpProject, "--gke-uri", gkeURI).CombinedOutput()
	if err != nil {
		nt.T.Fatalf("%s: %v", string(out), err)
	}
	tenant = "tenant-b"
	nt.T.Logf("Update RootSync to sync %s from a CSR repo", tenant)
	nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "%s"}}}`, tenant))
	nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(nomostest.RemoteRepoRootSha1Fn),
		nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: tenant}))
	validateTenant(nt, string(declared.RootReconciler), tenant)
	validateFWICredentials(nt, nomostest.DefaultRootReconcilerName, fwiAnnotationAbsent)

	// The CI jobs run the e2e test with service account `prober-runner@nomos-e2e-test1.iam.gserviceaccount.com`, which needs `roles/gkehub.admin` permission in the `cs-dev-hub` project.
	// Due to the domain restricted sharing policy, if fails to grant the permission to the external domain.
	// Disable the cross-project fleet workload identity authentication verification before the exemption is applied.
	//nt.T.Log("Register the cluster to a fleet in a different project")
	//crossProjectFleetProjectID := "cs-dev-hub"
	//out, err = exec.Command("gcloud", "beta", "container", "hub", "memberships", "register",
	//	fleetMembership, "--project", crossProjectFleetProjectID, "--gke-uri", gkeURI, "--enable-workload-identity").CombinedOutput()
	//if err != nil {
	//	nt.T.Fatalf("%s: %v", string(out), err)
	//}
	//tenant = "tenant-c"
	//nt.T.Logf("Update RootSync to sync %s from a CSR repo", tenant)
	//nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "%s"}}}`, tenant))
	//nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(nomostest.RemoteRepoRootSha1Fn),
	//	nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: tenant}))
	//validateTenant(nt, string(declared.RootReconciler), tenant)
	//validateFWICredentials(nt, nomostest.DefaultRootReconcilerName, fwiAnnotationExists)
	//
	//nt.T.Log("Unregister the cluster from the fleet in a different project")
	//out, err = exec.Command("gcloud", "beta", "container", "hub", "memberships",
	//	"unregister", fleetMembership, "--gke-uri", gkeURI, "--project", crossProjectFleetProjectID).CombinedOutput()
	//if err != nil {
	//	nt.T.Fatalf("%s: %v", string(out), err)
	//}
	//tenant = "tenant-d"
	//nt.T.Logf("Update RootSync to sync %s from a CSR repo", tenant)
	//nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "%s"}}}`, tenant))
	//nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(nomostest.RemoteRepoRootSha1Fn),
	//	nomostest.WithSyncDirectoryMap(map[types.NamespacedName]string{nomostest.DefaultRootRepoNamespacedName: tenant}))
	//validateTenant(nt, string(declared.RootReconciler), tenant)
	//validateFWICredentials(nt, nomostest.DefaultRootReconcilerName, fwiAnnotationAbsent)

	// Change the rs back so that it works in the shared test environment.
	nt.MustMergePatch(rs, fmt.Sprintf(`{"spec": {"git": {"dir": "acme", "branch": "main", "repo": "%s", "auth": "ssh","gcpServiceAccountEmail": "", "secretRef": {"name": "git-creds"}}, "sourceFormat": "hierarchy"}}`, origRepoURL))
}

// validateTenant validates if the tenant resources are created and managed by the reonciler.
func validateTenant(nt *nomostest.NT, reconcilerName, tenant string) {
	nt.T.Logf("Validate %s resources are created and managed by %s", tenant, reconcilerName)
	if err := nt.Validate(tenant, "", &corev1.Namespace{}, nomostest.HasAnnotation(metadata.ResourceManagerKey, reconcilerName)); err != nil {
		nt.T.Error(err)
	}
	if err := nt.Validate("deny-all", tenant, &networkingv1.NetworkPolicy{}, nomostest.HasAnnotation(metadata.ResourceManagerKey, reconcilerName)); err != nil {
		nt.T.Error(err)
	}
	if err := nt.Validate("tenant-admin", tenant, &rbacv1.Role{}, nomostest.HasAnnotation(metadata.ResourceManagerKey, reconcilerName)); err != nil {
		nt.T.Error(err)
	}
	if err := nt.Validate("tenant-admin-rolebinding", tenant, &rbacv1.RoleBinding{}, nomostest.HasAnnotation(metadata.ResourceManagerKey, reconcilerName)); err != nil {
		nt.T.Error(err)
	}
}

// validateFWICredentials validates whether the reconciler Pod manifests includes
// the FWI credentials annotation or not.
func validateFWICredentials(nt *nomostest.NT, reconcilerName string, validationFn func(pod corev1.Pod) error) {
	nt.T.Log("Validate reconciler Pod manifests")
	var podList = &corev1.PodList{}
	if err := nt.List(podList, client.InNamespace(configmanagement.ControllerNamespace), client.MatchingLabels{metadata.ReconcilerLabel: reconcilerName}); err != nil {
		nt.T.Fatal(err)
	}
	if len(podList.Items) != 1 {
		nt.T.Fatalf("expected only 1 Pod for the reconciler %s, but got %d", reconcilerName, len(podList.Items))
	}

	if err := validationFn(podList.Items[0]); err != nil {
		nt.T.Fatal(err)
	}
}

// fwiAnnotationAbsent validates if the Pod has the FWI credentials annotation.
func fwiAnnotationExists(pod corev1.Pod) error {
	if _, found := pod.GetAnnotations()[metadata.FleetWorkloadIdentityCredentials]; !found {
		return fmt.Errorf("object %s/%s does not have annotation %q", pod.GetNamespace(), pod.GetName(), metadata.FleetWorkloadIdentityCredentials)
	}
	return nil
}

// fwiAnnotationAbsent validates if the Pod doesn't have the FWI credentials annotation.
func fwiAnnotationAbsent(pod corev1.Pod) error {
	if _, found := pod.GetAnnotations()[metadata.FleetWorkloadIdentityCredentials]; found {
		return fmt.Errorf("object %s/%s has annotation %q", pod.GetNamespace(), pod.GetName(), metadata.FleetWorkloadIdentityCredentials)
	}
	return nil
}
