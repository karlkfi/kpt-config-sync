package e2e

import (
	"fmt"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This file includes tests for KCC resources.
// The test applies KCC resources and verifies the GCP resources
// are created successfully.
// It then deletes KCC resources and verifies the GCP resources
// are removed successfully.
func TestKCCResources(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.KccTest)

	// Namespace foo holds the KCC resources.
	nt.Root.Add("acme/namespaces/foo/ns.yaml",
		fake.NamespaceObject("foo",
			// Annotate the namespace to create GCP resources in the project "jingfang-fishfood".
			core.Annotation("cnrm.cloud.google.com/project-id", "jingfang-fishfood")))
	nt.Root.CommitAndPush("add Namespace for holding KCC resources")
	nt.WaitForRepoSyncs()

	// Add KCC resources
	enablePubSub := []byte(`
apiVersion: serviceusage.cnrm.cloud.google.com/v1beta1
kind: Service
metadata:
  name: pubsub.googleapis.com
  namespace: foo
  annotations:
    cnrm.cloud.google.com/deletion-policy: "abandon"
`)
	pubsubTopic := []byte(`
apiVersion: pubsub.cnrm.cloud.google.com/v1beta1
kind: PubSubTopic
metadata:
  labels:
    environment: staging
  name: test-cs
  namespace: foo
`)
	pubsubKey := []byte(`
apiVersion: iam.cnrm.cloud.google.com/v1beta1
kind: IAMServiceAccountKey
metadata:
  name: pubsub-key
  namespace: foo
spec:
  publicKeyType: TYPE_X509_PEM_FILE
  keyAlgorithm: KEY_ALG_RSA_2048
  privateKeyType: TYPE_GOOGLE_CREDENTIALS_FILE
  serviceAccountRef:
    name: pubsub-app
`)
	policy := []byte(`
apiVersion: iam.cnrm.cloud.google.com/v1beta1
kind: IAMPolicyMember
metadata:
  name: policy-member-binding
  namespace: foo
spec:
  member: serviceAccount:pubsub-app@jingfang-fishfood.iam.gserviceaccount.com
  role: roles/pubsub.subscriber
  resourceRef:
    apiVersion: resourcemanager.cnrm.cloud.google.com/v1beta1
    kind: Project
    external: projects/jingfang-fishfood
`)
	serviceAccount := []byte(`
apiVersion: iam.cnrm.cloud.google.com/v1beta1
kind: IAMServiceAccount
metadata:
  name: pubsub-app
  namespace: foo
spec:
  displayName: Service account for PubSub example
`)
	subscription := []byte(`
apiVersion: pubsub.cnrm.cloud.google.com/v1beta1
kind: PubSubSubscription
metadata:
  name: test-cs-read
  namespace: foo
spec:
  topicRef:
    name: test-cs
`)
	nt.Root.AddFile("acme/namespaces/foo/enable-pubsub.yaml", enablePubSub)
	nt.Root.AddFile("acme/namespaces/foo/pubsub-topic.yaml", pubsubTopic)
	nt.Root.AddFile("acme/namespaces/foo/pubsub-key.yaml", pubsubKey)
	nt.Root.AddFile("acme/namespaces/foo/service-account-policy.yaml", policy)
	nt.Root.AddFile("acme/namespaces/foo/service-account.yaml", serviceAccount)
	nt.Root.AddFile("acme/namespaces/foo/subscription.yaml", subscription)
	nt.Root.CommitAndPush("add KCC resources")
	nt.WaitForRepoSyncs()

	// Verify that the GCP resources are created.
	gvkPubSubTopic := schema.GroupVersionKind{
		Group:   "pubsub.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "PubSubTopic",
	}
	gvkPubSubSubscription := schema.GroupVersionKind{
		Group:   "pubsub.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "PubSubSubscription",
	}
	gvkServiceAccount := schema.GroupVersionKind{
		Group:   "iam.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "IAMServiceAccount",
	}
	gvkPolicyMember := schema.GroupVersionKind{
		Group:   "iam.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "IAMPolicyMember",
	}
	validateKCCResourceReady(nt, gvkPubSubTopic, "test-cs", "foo")
	validateKCCResourceReady(nt, gvkPubSubSubscription, "test-cs-read", "foo")
	validateKCCResourceReady(nt, gvkServiceAccount, "pubsub-app", "foo")
	validateKCCResourceReady(nt, gvkPolicyMember, "policy-member-binding", "foo")

	// Remove the kcc resources
	nt.Root.Remove("acme/namespaces/foo/enable-pubsub.yaml")
	nt.Root.Remove("acme/namespaces/foo/pubsub-topic.yaml")
	nt.Root.Remove("acme/namespaces/foo/pubsub-key.yaml")
	nt.Root.Remove("acme/namespaces/foo/service-account-policy.yaml")
	nt.Root.Remove("acme/namespaces/foo/service-account.yaml")
	nt.Root.Remove("acme/namespaces/foo/subscription.yaml")
	nt.Root.Remove("acme/namespaces/foo/ns.yaml")
	nt.Root.CommitAndPush("remove KCC resources")
	nt.WaitForRepoSyncs()

	// Verify that the GCP resources are removed.
	validateKCCResourceNotFound(nt, gvkPubSubTopic, "test-cs", "foo")
	validateKCCResourceNotFound(nt, gvkPubSubSubscription, "test-cs-read", "foo")
	validateKCCResourceNotFound(nt, gvkServiceAccount, "pubsub-app", "foo")
	validateKCCResourceNotFound(nt, gvkPolicyMember, "policy-member-binding", "foo")

}

// This file includes tests for KCC resources from a cloud source repository.
// The test applies KCC resources and verifies the GCP resources
// are created successfully.
// It then deletes KCC resources and verifies the GCP resources
// are removed successfully.
func TestKCCResourcesOnCSR(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.KccTest)
	rs := fake.RootSyncObjectV1Beta1()
	nt.T.Log("sync to the kcc resources from a CSR repo")
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "kcc", "branch": "main", "repo": "https://source.developers.google.com/p/stolos-dev/r/configsync-ci-cc", "auth": "gcpserviceaccount","gcpServiceAccountEmail": "e2e-test-csr-reader@stolos-dev.iam.gserviceaccount.com", "secretRef": {"name": ""}}, "sourceFormat": "unstructured"}}`)

	sha1Fn := func(nt *nomostest.NT) (string, error) {
		rs = &v1beta1.RootSync{}
		if err := nt.Get("root-sync", configmanagement.ControllerNamespace, rs); err != nil {
			return "", err
		}
		return rs.Status.LastSyncedCommit, nil
	}
	nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(sha1Fn), nomostest.WithSyncDirectory("kcc"))

	// Verify that the GCP resources are created.
	gvkPubSubTopic := schema.GroupVersionKind{
		Group:   "pubsub.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "PubSubTopic",
	}
	gvkPubSubSubscription := schema.GroupVersionKind{
		Group:   "pubsub.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "PubSubSubscription",
	}
	gvkServiceAccount := schema.GroupVersionKind{
		Group:   "iam.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "IAMServiceAccount",
	}
	gvkPolicyMember := schema.GroupVersionKind{
		Group:   "iam.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "IAMPolicyMember",
	}
	validateKCCResourceReady(nt, gvkPubSubTopic, "test-cs", "foo")
	validateKCCResourceReady(nt, gvkPubSubSubscription, "test-cs-read", "foo")
	validateKCCResourceReady(nt, gvkServiceAccount, "pubsub-app", "foo")
	validateKCCResourceReady(nt, gvkPolicyMember, "policy-member-binding", "foo")

	// Remove the kcc resources
	nt.T.Log("sync to an empty directory from a CSR repo")
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "kcc-empty"}}}`)
	nt.WaitForRepoSyncs(nomostest.WithRootSha1Func(sha1Fn), nomostest.WithSyncDirectory("kcc-empty"))

	// Verify that the GCP resources are removed.
	validateKCCResourceNotFound(nt, gvkPubSubTopic, "test-cs", "foo")
	validateKCCResourceNotFound(nt, gvkPubSubSubscription, "test-cs-read", "foo")
	validateKCCResourceNotFound(nt, gvkServiceAccount, "pubsub-app", "foo")
	validateKCCResourceNotFound(nt, gvkPolicyMember, "policy-member-binding", "foo")

	// Change the rs back so that it works in the shared test environment.
	nt.MustMergePatch(rs, `{"spec": {"git": {"dir": "acme", "branch": "main", "repo": "git@test-git-server.config-management-system-test:/git-server/repos/sot.git", "auth": "ssh","gcpServiceAccountEmail": "", "secretRef": {"name": "git-creds"}}, "sourceFormat": "hierarchy"}}`)
}

func validateKCCResourceReady(nt *nomostest.NT, gvk schema.GroupVersionKind, name, namespace string) {
	nomostest.Wait(nt.T, fmt.Sprintf("wait for kcc resources %q %v to be ready", name, gvk),
		nt.DefaultWaitTimeout, func() error {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(gvk)
			return nt.Validate(name, namespace, u, kccResourceReady)
		})
}

func kccResourceReady(o client.Object) error {
	u := o.(*unstructured.Unstructured)
	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil || !found || len(conditions) == 0 {
		return fmt.Errorf(".status.conditions not found %v", err)
	}
	condition := (conditions[0]).(map[string]interface{})
	if condition["type"] != "Ready" || condition["status"] != "True" {
		return fmt.Errorf("resource is not ready %v", condition)
	}
	return nil
}

func validateKCCResourceNotFound(nt *nomostest.NT, gvk schema.GroupVersionKind, name, namespace string) {
	nomostest.Wait(nt.T, fmt.Sprintf("wait for %q %v to terminate", name, gvk),
		nt.DefaultWaitTimeout, func() error {
			u := &unstructured.Unstructured{}
			u.SetGroupVersionKind(gvk)
			return nt.ValidateNotFound(name, namespace, u)
		})
}
