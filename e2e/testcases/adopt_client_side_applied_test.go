package e2e

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/e2e/nomostest"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestAdoptClientSideAppliedResource(t *testing.T) {
	nt := nomostest.New(t)

	// Declare a ClusterRole and `kubectl apply -f` it to the cluster.
	nsViewerName := "ns-viewer"
	nsViewer := fake.ClusterRoleObject(core.Name(nsViewerName),
		core.Label("permissions", "viewer"))
	nsViewer.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"namespaces"},
		Verbs:     []string{"get", "list"},
	}}

	nt.RootRepos[configsync.RootSyncName].Add("ns-viewer-client-side-applied.yaml", nsViewer)
	nt.MustKubectl("apply", "-f", filepath.Join(nt.RootRepos[configsync.RootSyncName].Root, "ns-viewer-client-side-applied.yaml"))

	// Validate the ClusterRole exist.
	err := nt.Validate(nsViewerName, "", &rbacv1.ClusterRole{})
	if err != nil {
		nt.T.Fatal(err)
	}

	// Add the ClusterRole and let ConfigSync to sync it.
	nsViewer.Rules = []rbacv1.PolicyRule{{
		APIGroups: []string{""},
		Resources: []string{"namespaces"},
		Verbs:     []string{"get"},
	}}
	nt.RootRepos[configsync.RootSyncName].Add("acme/cluster/ns-viewer-cr.yaml", nsViewer)
	nt.RootRepos[configsync.RootSyncName].CommitAndPush("add namespace-viewer ClusterRole")
	nt.WaitForRepoSyncs()

	// Validate the ClusterRole exist and the Rules are the same as the one
	// in "acme/cluster/ns-viewer-cr.yaml".
	role := &rbacv1.ClusterRole{}
	err = nt.Validate(nsViewerName, "", role)
	if err != nil {
		nt.T.Fatal(err)
	}

	if diff := cmp.Diff(role.Rules[0].Verbs, []string{"get"}); diff != "" {
		nt.T.Errorf(diff)
	}
}
