package e2e

import (
	"fmt"
	"testing"

	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/e2e/nomostest/ntopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	label = "stress-test"
	value = "enabled"
)

func Test_10000_Objects(t *testing.T) {
	// Expect this test to take ~1 hour to complete (i.e. run with --timeout=1h).
	// We only add about 5 objects per second. Lower n if you're just sanity testing.
	n := 10000
	nt := nomostest.New(t, ntopts.RemoteCluster(t), ntopts.Unstructured)

	nt.Root.Add("acme/ns.yaml", fake.NamespaceObject(namespace(1)))
	for i := 0; i < n; i++ {
		nt.Root.Add(path(i), configMap(1, i))
	}
	nt.Root.CommitAndPush("add ConfigMaps")
	nt.WaitForRepoSyncs()
}

func Test_100_x_100_Objects(t *testing.T) {
	// Expect this test to take ~30 minutes to complete (i.e. run with --timeout=30m).
	// Fails (after a long time) if run on a cluster which is unable to support
	// 100 Namespace repos.
	nNamespaces := 100
	nConfigMaps := 100

	opts := []ntopts.Opt{ntopts.Unstructured, ntopts.RemoteCluster(t), ntopts.WithDelegatedControl}
	for i := 0; i < nNamespaces; i++ {
		opts = append(opts, ntopts.NamespaceRepo(namespace(i)))
	}
	nt := nomostest.New(t, opts...)

	for i := 0; i < nNamespaces; i++ {
		repo := nt.NonRootRepos[namespace(i)]
		for j := 0; j < nConfigMaps; j++ {
			repo.Add(path(j), configMap(i, j))
		}
		repo.CommitAndPush("Add ConfigMaps")
	}
	nt.WaitForRepoSyncs()

	list := &corev1.ConfigMapList{}
	err := nt.List(list, client.MatchingLabels{label: value})
	if err != nil {
		t.Fatal(err)
	}

	if len(list.Items) != nNamespaces*nConfigMaps {
		t.Errorf("got %d ConfigMaps, want %d", len(list.Items), nNamespaces*nConfigMaps)
	}
}

func namespace(i int) string {
	return fmt.Sprintf("foo-%d", i)
}

func name(j int) string {
	return fmt.Sprintf("configmap-%d", j)
}

func path(j int) string {
	return fmt.Sprintf("acme/%s.yaml", name(j))
}

func configMap(i, j int) *corev1.ConfigMap {
	cm := fake.ConfigMapObject(core.Name(name(j)), core.Namespace(namespace(i)), core.Label(label, value))
	cm.Data = map[string]string{
		fmt.Sprintf("foo-%d", i): fmt.Sprintf("bar-%d", j),
	}
	return cm
}
