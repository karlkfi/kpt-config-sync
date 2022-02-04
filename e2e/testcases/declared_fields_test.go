package e2e

import (
	"testing"
	"time"

	"github.com/google/nomos/pkg/api/configsync"
	corev1 "k8s.io/api/core/v1"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDeclaredFieldsPod(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.RootRepos[configsync.RootSyncName].Add("acme/ns.yaml", namespace)
	// We use literal YAML here instead of an object as:
	// 1) If we used a literal struct the protocol field would implicitly be added.
	// 2) It's really annoying to specify this as Unstructureds.
	nt.RootRepos[configsync.RootSyncName].AddFile("acme/pod.yaml", []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  namespace: bookstore
spec:
  containers:
  - image: nginx:1.7.9
    name: nginx
    ports:
    - containerPort: 80
`))
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add pod missing protocol from port")
	nt.WaitForRepoSyncs()

	err := nt.Validate("nginx", namespace.Name, &corev1.Pod{})
	if err != nil {
		nt.T.Fatal(err)
	}

	nt.RootRepos[configsync.RootSyncName].Remove("acme/pod.yaml")
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("Remove the pod")
	nt.WaitForRepoSyncs()

	_, err = nomostest.Retry(60*time.Second, func() error {
		return nt.ValidateNotFound("nginx", namespace.Name, &corev1.Pod{})
	})
	if err != nil {
		nt.T.Fatal(err)
	}
}
