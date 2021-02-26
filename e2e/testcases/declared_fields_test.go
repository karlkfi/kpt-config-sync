package e2e

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDeclaredFieldsPod(t *testing.T) {
	nt := nomostest.New(t, ntopts.SkipMonoRepo, ntopts.Unstructured)

	namespace := fake.NamespaceObject("bookstore")
	nt.Root.Add("acme/ns.yaml", namespace)
	// We use literal YAML here instead of an object as:
	// 1) If we used a literal struct the protocol field would implicitly be added.
	// 2) It's really annoying to specify this as Unstructureds.
	nt.Root.AddFile("acme/pod.yaml", []byte(`
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
	nt.Root.CommitAndPush("add pod missing protocol from port")
	nt.WaitForRepoSyncs()

	pod := &corev1.Pod{}
	err := nt.Validate("nginx", namespace.Name, pod)
	if err != nil {
		t.Fatal(err)
	}

	// TODO(b/184764581): This should be deleted once b/184764581 is fixed.
	cleanup(nt, namespace, pod)
}

// cleanup gracefully deletes the test resources (namespace and pod).
// Due to b/184764581, the pod and the namespace are stuck in the terminating state.
// To delete them gracefully, we unmanage them from the repo and manually delete them.
func cleanup(nt *nomostest.NT, ns *corev1.Namespace, pod *corev1.Pod) {
	ns.Annotations[v1.ResourceManagementKey] = v1.ResourceManagementDisabled
	nt.Root.Add("acme/ns.yaml", ns)
	pod = fake.PodObject(pod.Name, pod.Spec.Containers, core.Namespace(pod.Namespace),
		core.Annotation(v1.ResourceManagementKey, v1.ResourceManagementDisabled))
	nt.Root.Add("acme/pod.yaml", pod)
	nt.Root.CommitAndPush("unmanage the pod and the namespace")
	nt.WaitForRepoSyncs()

	if err := nt.Delete(ns); err != nil {
		nt.T.Fatal(err)
	}
	nomostest.WaitToTerminate(nt, kinds.Namespace(), ns.Name, "")
	if err := nt.ValidateNotFound(pod.Name, pod.Namespace, &corev1.Pod{}); err != nil {
		nt.T.Fatal(err)
	}
}
