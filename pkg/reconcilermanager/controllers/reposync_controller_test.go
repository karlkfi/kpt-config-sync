package controllers

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	uid           = types.UID("1234")
	auth          = "ssh"
	branch        = "1.0.0"
	updatedBranch = "2.0.0"

	reposyncReqNamespace = "bookinfo"
	reposyncKind         = "RepoSync"
	reposyncName         = "repo-sync"
	reposyncRepo         = "https://github.com/test/reposync/csp-config-management/"
	reposyncDir          = "foo-corp"
)

func repoSync(rev string, opts ...core.MetaMutator) v1.RepoSync {
	result := fake.RepoSyncObject(opts...)
	result.Spec.Git = v1.Git{
		Repo:     reposyncRepo,
		Revision: rev,
		Dir:      reposyncDir,
		Auth:     auth,
	}
	return *result
}

func TestRepoSyncMutateConfigMap(t *testing.T) {
	testCases := []struct {
		name            string
		repoSync        v1.RepoSync
		actualConfigMap *corev1.ConfigMap
		wantConfigMap   *corev1.ConfigMap
	}{
		{
			name: "ConfigMap created",
			repoSync: repoSync(
				"1.0.0",
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				repoSyncReconcilerPrefix+reposyncReqNamespace,
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				repoSyncReconcilerPrefix+reposyncReqNamespace,
				configMapData(branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
		},
		{
			name: "ConfigMap updated with revision number",
			repoSync: repoSync(
				"2.0.0",
				core.Name("repo-sync"),
				core.Namespace("bookinfo"),
				core.UID(uid),
			),
			actualConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				repoSyncReconcilerPrefix+reposyncReqNamespace,
				configMapData(branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid)),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				repoSyncReconcilerPrefix+reposyncReqNamespace,
				configMapData(updatedBranch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mutateRepoSyncConfigMap(tc.repoSync, tc.actualConfigMap)
			if !cmp.Equal(tc.actualConfigMap, tc.wantConfigMap) {
				t.Errorf("got: %v\nwant: %v", spew.Sdump(tc.actualConfigMap), spew.Sdump(tc.wantConfigMap))
			}
		})
	}
}

func TestRepoSyncMutateDeployment(t *testing.T) {
	testCases := []struct {
		name             string
		repoSync         v1.RepoSync
		actualDeployment *appsv1.Deployment
		wantDeployment   *appsv1.Deployment
	}{
		{
			name: "Deployment created",
			repoSync: repoSync(
				"1.0.0",
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				repoSyncReconcilerPrefix+reposyncReqNamespace,
				"git-sync"),
			wantDeployment: deploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				repoSyncReconcilerPrefix+reposyncReqNamespace,
				"git-sync",
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mutateRepoSyncDeployment(tc.repoSync, tc.actualDeployment)
			if !cmp.Equal(tc.actualDeployment, tc.wantDeployment) {
				t.Errorf("\ngot:  %v\nwant: %v", spew.Sdump(tc.actualDeployment), spew.Sdump(tc.wantDeployment))
			}
		})
	}
}
