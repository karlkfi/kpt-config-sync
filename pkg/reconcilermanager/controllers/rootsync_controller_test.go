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

func deploymentWithEnvFrom(namespace, name, containerName string, opts ...core.MetaMutator) *appsv1.Deployment {
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
				rootsyncReqNamespace,
				rootsyncName,
			),
			wantConfigMap: configMapWithData(
				rootsyncReqNamespace,
				rootsyncName,
				configMapData(branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
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
				rootsyncReqNamespace,
				rootsyncName,
				configMapData(branch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid)),
			),
			wantConfigMap: configMapWithData(
				rootsyncReqNamespace,
				rootsyncName,
				configMapData(updatedBranch, rootsyncRepo),
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mutateRootSyncConfigMap(tc.rootSync, tc.actualConfigMap)
			if !cmp.Equal(tc.actualConfigMap, tc.wantConfigMap) {
				t.Errorf("\ngot:  %v\nwant: %v", spew.Sdump(tc.actualConfigMap), spew.Sdump(tc.wantConfigMap))
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
				rootsyncReqNamespace,
				rootSyncReconcilerName,
				"git-sync"),
			wantDeployment: deploymentWithEnvFrom(
				rootsyncReqNamespace,
				rootSyncReconcilerName,
				"git-sync",
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mutateRootSyncDeployment(tc.rootSync, tc.actualDeployment)
			if !cmp.Equal(tc.actualDeployment, tc.wantDeployment) {
				t.Errorf("\ngot:  %v\nwant: %v", spew.Sdump(tc.actualDeployment), spew.Sdump(tc.wantDeployment))
			}
		})
	}
}
