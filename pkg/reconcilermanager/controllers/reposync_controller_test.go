package controllers

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
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

	unsupportedConfigMap = "xyz"
	unsupportedContainer = "abc"
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

type configMapRef struct {
	name     string
	optional *bool
}

func nsGitSyncConfigMap(containerName string, configmap configMapRef) map[string][]configMapRef {
	result := make(map[string][]configMapRef)
	result[containerName] = []configMapRef{configmap}
	return result
}

// nsDeploymentWithEnvFrom returns appsv1.Deployment
// containerConfigMap contains map of container name and their respective configmaps.
func nsDeploymentWithEnvFrom(namespace, name string, containerConfigMap map[string][]configMapRef, opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = namespace
	result.Name = buildRepoSyncName(name)

	var container []corev1.Container
	for cntrName, cms := range containerConfigMap {
		cntr := fake.ContainerObject(cntrName)
		var eFromSource []corev1.EnvFromSource
		for _, cm := range cms {
			eFromSource = append(eFromSource, envFromSource(name, cm)) // buildRepoSyncName(reposyncReqNamespace)
		}
		cntr.EnvFrom = append(cntr.EnvFrom, eFromSource...)
		container = append(container, *cntr)
	}
	result.Spec.Template.Spec = corev1.PodSpec{
		Containers: container,
	}
	return result
}

func envFromSource(name string, configMap configMapRef) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: buildRepoSyncName(name, configMap.name),
			},
			Optional: configMap.optional,
		},
	}
}

func TestRepoSyncMutateConfigMap(t *testing.T) {
	testCases := []struct {
		name            string
		repoSync        v1.RepoSync
		actualConfigMap *corev1.ConfigMap
		wantConfigMap   *corev1.ConfigMap
		wantErr         bool
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
				buildRepoSyncName(reposyncReqNamespace, gitSync),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, gitSync),
				gitSyncData(branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap updated with revision number",
			repoSync: repoSync(
				"2.0.0",
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, gitSync),
				gitSyncData(branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid)),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, gitSync),
				gitSyncData(updatedBranch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap mutate failed, Unsupported ConfigMap",
			repoSync: repoSync(
				"1.0.0",
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, unsupportedConfigMap),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace, unsupportedConfigMap),
				gitSyncData(branch, reposyncRepo),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mutateRepoSyncConfigMap(tc.repoSync, tc.actualConfigMap)
			if tc.wantErr && err == nil {
				t.Errorf("mutateRepoSyncConfigMap() got error: %q, want error: %t", err, tc.wantErr)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRepoSyncConfigMap() got error: %q, want error: %t", err, tc.wantErr)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.actualConfigMap, tc.wantConfigMap); diff != "" {
					t.Errorf("mutateRepoSyncConfigMap() got diff: %v\nwant: nil", diff)
				}
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
		wantErr          bool
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
				buildRepoSyncName(reposyncReqNamespace),
				"git-sync"),
			wantDeployment: nsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				reposyncReqNamespace,
				nsGitSyncConfigMap(buildRepoSyncName(reposyncReqNamespace, gitSync), configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: false,
		},
		{
			name: "Deployment failed, Unsupported container",
			repoSync: repoSync(
				"1.0.0",
				core.Name(reposyncName),
				core.Namespace(reposyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				buildRepoSyncName(reposyncReqNamespace),
				unsupportedContainer),
			wantDeployment: nsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				reposyncReqNamespace,
				nsGitSyncConfigMap(buildRepoSyncName(reposyncReqNamespace, gitSync), configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				core.OwnerReference(ownerReference(reposyncKind, reposyncName, uid))),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mutateRepoSyncDeployment(tc.repoSync, tc.actualDeployment)
			if tc.wantErr && err == nil {
				t.Errorf("mutateRepoSyncDeployment() got error: %q, want error: %t", err, tc.wantErr)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRepoSyncDeployment() got error: %q, want error: %t", err, tc.wantErr)
			}
			if !tc.wantErr {
				diff := cmp.Diff(tc.actualDeployment, tc.wantDeployment)
				if diff != "" {
					t.Errorf("mutateRepoSyncDeployment() got diff: %v\nwant: nil", diff)
				}
			}
		})
	}
}
