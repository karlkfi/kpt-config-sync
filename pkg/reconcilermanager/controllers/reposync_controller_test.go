package controllers

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	reqNamespace    = "bookinfo"
	rskind          = "RepoSync"
	rsName          = "repo-sync"
	rsUID           = types.UID("1234")
	rsRepo          = "https://github.com/GoogleCloudPlatform/csp-config-management/"
	rsBranch        = "1.0.0"
	rsAuth          = "ssh"
	rsDir           = "foo-corp"
	rsUpdatedBranch = "2.0.0"
)

func repoSync(rev string, opts ...core.MetaMutator) v1.RepoSync {
	result := fake.RepoSyncObject(opts...)
	result.Spec.Git = v1.Git{
		Repo:     rsRepo,
		Revision: rev,
		Dir:      rsDir,
		Auth:     rsAuth,
	}
	return *result
}

func configMap(opts ...core.MetaMutator) *corev1.ConfigMap {
	result := fake.ConfigMapObject(opts...)
	result.Namespace = v1.NSConfigManagementSystem
	result.Name = repoSyncReconcilerPrefix + reqNamespace
	return result
}

func configMapWithData(data map[string]string, opts ...core.MetaMutator) *corev1.ConfigMap {
	result := fake.ConfigMapObject(opts...)
	result.Namespace = v1.NSConfigManagementSystem
	result.Name = repoSyncReconcilerPrefix + reqNamespace
	result.Data = data
	return result
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
				core.Name(rsName),
				core.Namespace(reqNamespace),
				core.UID(rsUID),
			),
			actualConfigMap: configMap(),
			wantConfigMap: configMapWithData(
				configMapData(rsBranch, rsRepo),
				core.OwnerReference(ownerReference(rskind, rsName, rsUID))),
		},
		{
			name: "ConfigMap updated with revision number",
			repoSync: repoSync(
				"2.0.0",
				core.Name("repo-sync"),
				core.Namespace("bookinfo"),
				core.UID(rsUID),
			),
			actualConfigMap: configMapWithData(
				configMapData(rsBranch, rsRepo),
				core.OwnerReference(ownerReference(rskind, rsName, rsUID)),
			),
			wantConfigMap: configMapWithData(
				configMapData(rsUpdatedBranch, rsRepo),
				core.OwnerReference(ownerReference(rskind, rsName, rsUID))),
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

func deployment(containerName string, opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = v1.NSConfigManagementSystem
	result.Name = repoSyncReconcilerPrefix + reqNamespace
	result.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			*fake.ContainerObject(containerName),
		},
	}
	return result
}

func deploymentWithEnvFrom(containerName string, ns string, opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = v1.NSConfigManagementSystem
	result.Name = repoSyncReconcilerPrefix + reqNamespace

	container := fake.ContainerObject(containerName)
	container.EnvFrom = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: repoSyncReconcilerPrefix + ns,
				},
			},
		},
	}

	result.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			*container,
		},
	}
	return result
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
				core.Name(rsName),
				core.Namespace(reqNamespace),
				core.UID(rsUID),
			),
			actualDeployment: deployment("git-sync"),
			wantDeployment: deploymentWithEnvFrom(
				"git-sync",
				reqNamespace,
				core.OwnerReference(ownerReference(rskind, rsName, rsUID))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mutateRepoSyncDeployment(tc.repoSync, tc.actualDeployment)
			if !cmp.Equal(tc.actualDeployment, tc.wantDeployment) {
				t.Errorf("got: %v\nwant: %v", spew.Sdump(tc.actualDeployment), spew.Sdump(tc.wantDeployment))
			}
		})
	}
}
