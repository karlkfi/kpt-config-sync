package controllers

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/utils/pointer"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	rootsyncReqNamespace = "config-management-system"
	rootsyncKind         = "RootSync"
	rootsyncName         = "root-sync"
	rootsyncRepo         = "https://github.com/test/rootsync/csp-config-management/"
	rootsyncDir          = "baz-corp"
)

func configMap(namespace, name string, opts ...core.MetaMutator) *corev1.ConfigMap {
	result := fake.ConfigMapObject(opts...)
	result.Namespace = namespace
	result.Name = name
	return result
}

func configMapWithData(namespace, name string, data map[string]string, opts ...core.MetaMutator) *corev1.ConfigMap {
	result := fake.ConfigMapObject(opts...)
	result.Namespace = namespace
	result.Name = name
	result.Data = data
	return result
}

func deployment(namespace, name string, containerName string, opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = namespace
	result.Name = name
	result.Spec.Template.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			*fake.ContainerObject(containerName),
		},
	}
	return result
}

func rootDeploymentWithEnvFrom(namespace, name, containerName string, opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = namespace
	result.Name = name

	container := fake.ContainerObject(containerName)
	container.EnvFrom = []corev1.EnvFromSource{
		{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Optional: pointer.BoolPtr(false),
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

func rootSync(rev string, opts ...core.MetaMutator) v1.RootSync {
	result := fake.RootSyncObject(opts...)
	result.Spec.Git = v1.Git{
		Repo:     rootsyncRepo,
		Revision: rev,
		Dir:      rootsyncDir,
		Auth:     auth,
	}
	return *result
}

func TestRootSyncMutateConfigMap(t *testing.T) {
	testCases := []struct {
		name            string
		rootSync        v1.RootSync
		actualConfigMap *corev1.ConfigMap
		wantConfigMap   *corev1.ConfigMap
		wantErr         bool
	}{
		{
			name: "ConfigMap created",
			rootSync: rootSync(
				"1.0.0",
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncData(branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap updated with revision number",
			rootSync: rootSync(
				"2.0.0",
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncData(branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid)),
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncData(updatedBranch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: false,
		},
		{
			name: "ConfigMap mutate failed, Unsupported ConfigMap",
			rootSync: rootSync(
				"1.0.0",
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualConfigMap: configMap(
				v1.NSConfigManagementSystem,
				unsupportedConfigMap,
			),
			wantConfigMap: configMapWithData(
				v1.NSConfigManagementSystem,
				unsupportedConfigMap,
				gitSyncData(branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mutateRootSyncConfigMap(tc.rootSync, tc.actualConfigMap)
			if tc.wantErr && err == nil {
				t.Errorf("mutateRootSyncConfigMap() got error: %q, want error", err)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRootSyncConfigMap() got error: %q, want error: nil", err)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.actualConfigMap, tc.wantConfigMap); diff != "" {
					t.Errorf("mutateRootSyncConfigMap() got diff: %v\nwant: nil", diff)
				}
			}
		})
	}
}

func TestRootSyncMutateDeployment(t *testing.T) {
	testCases := []struct {
		name             string
		rootSync         v1.RootSync
		actualDeployment *appsv1.Deployment
		wantDeployment   *appsv1.Deployment
		wantErr          bool
	}{
		{
			name: "Deployment created",
			rootSync: rootSync(
				"1.0.0",
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSync),
			wantDeployment: rootDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSync,
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: false,
		},
		{
			name: "Deployment failed, Unsupported container",
			rootSync: rootSync(
				"1.0.0",
				core.Name(rootsyncName),
				core.Namespace(rootsyncReqNamespace),
				core.UID(uid),
			),
			actualDeployment: deployment(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				unsupportedContainer),
			wantDeployment: rootDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				gitSync,
				gitSync,
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mutateRootSyncDeployment(tc.rootSync, tc.actualDeployment)
			if tc.wantErr && err == nil {
				t.Errorf("mutateRepoSyncDeployment() got error: %q, want error", err)
			} else if !tc.wantErr && err != nil {
				t.Errorf("mutateRepoSyncDeployment() got error: %q, want error: nil", err)
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
