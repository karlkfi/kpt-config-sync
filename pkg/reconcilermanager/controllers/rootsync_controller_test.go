package controllers

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/pointer"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	syncerFake "github.com/google/nomos/pkg/syncer/testing/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	secretAuth           = "ssh"
	rootsyncReqNamespace = "config-management-system"
	rootsyncKind         = "RootSync"
	rootsyncName         = "root-sync"
	rootsyncRepo         = "https://github.com/test/rootsync/csp-config-management/"
	rootsyncDir          = "baz-corp"

	// Hash of all configmap.data created by Root Reconciler.
	rsAnnotation = "49d0d5da30e10d1759e945f1b9ed61c2"
	// Updated hash of all configmap.data updated by Root Reconciler.
	rsUpdatedAnnotation = "d92e449392ac477b84e07e8ea88ed6c5"

	rootsyncSSHKey = "root-ssh-key"
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

func secretData(t *testing.T, auth string) map[string][]byte {
	t.Helper()
	key, err := json.Marshal("test-key")
	if err != nil {
		t.Fatalf("failed to marshal test key: %v", err)
	}
	return map[string][]byte{
		auth: key,
	}
}

func secret(t *testing.T, name, auth string, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, auth)
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

func rsDeploymentAnnotation() map[string]string {
	return map[string]string{
		"configmanagement.gke.io/configmap": rsAnnotation,
	}
}

func rsDeploymentUpdatedAnnotation() map[string]string {
	return map[string]string{
		"configmanagement.gke.io/configmap": rsUpdatedAnnotation,
	}
}

// rsDeploymentWithEnvFrom returns appsv1.Deployment
// containerConfigMap contains map of container name and their respective configmaps.
func rsDeploymentWithEnvFrom(namespace, name string,
	containerConfigMap map[string][]configMapRef,
	annotation map[string]string,
	opts ...core.MetaMutator) *appsv1.Deployment {
	result := fake.DeploymentObject(opts...)
	result.Namespace = namespace
	result.Name = name
	result.Spec.Template.Annotations = annotation

	result.Spec.Template.Spec = corev1.PodSpec{
		Containers: importerContainer(name, containerConfigMap),
	}
	return result
}

func importerContainer(name string, containerConfigMap map[string][]configMapRef) []corev1.Container {
	var container []corev1.Container
	for cntrName, cms := range containerConfigMap {
		cntr := fake.ContainerObject(cntrName)
		var eFromSource []corev1.EnvFromSource
		for _, cm := range cms {
			eFromSource = append(eFromSource, rsEnvFromSource(cm))
		}
		cntr.EnvFrom = append(cntr.EnvFrom, eFromSource...)
		container = append(container, *cntr)
	}
	return container
}

func rsEnvFromSource(configMap configMapRef) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: buildRootSyncName(configMap.name),
			},
			Optional: configMap.optional,
		},
	}
}

func rootSync(rev string, opts ...core.MetaMutator) *v1.RootSync {
	result := fake.RootSyncObject(opts...)
	result.Spec.Git = v1.Git{
		Repo:      rootsyncRepo,
		Revision:  rev,
		Dir:       rootsyncDir,
		Auth:      auth,
		SecretRef: v1.SecretReference{Name: rootsyncSSHKey},
	}
	return result
}

func TestRootSyncMutateConfigMap(t *testing.T) {
	testCases := []struct {
		name            string
		rootSync        *v1.RootSync
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
			_, err := mutateRootSyncConfigMap(*tc.rootSync, tc.actualConfigMap)
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
		rootSync         *v1.RootSync
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
			wantDeployment: rsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				buildRootSyncName(gitSync),
				gitSyncConfigMap(gitSync, configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				map[string]string{"configmanagement.gke.io/configmap": "31323334"},
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
			wantDeployment: rsDeploymentWithEnvFrom(
				v1.NSConfigManagementSystem,
				gitSync,
				gitSyncConfigMap(gitSync, configMapRef{
					name:     gitSync,
					optional: pointer.BoolPtr(false),
				}),
				nil,
				core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, uid))),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := mutateRootSyncDeployment(*tc.rootSync, tc.actualDeployment, []byte("1234"))
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

func TestRootSyncReconciler(t *testing.T) {
	// Mock out parseDeployment for testing.
	rsParseDeployment = func(de *appsv1.Deployment) error {
		de.Spec = appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: importer},
						{Name: gitSync},
						{Name: fsWatcher},
					},
				},
			},
		}
		return nil
	}

	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := appsv1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}

	rsResource := rootSync(branch, core.Name(rootsyncName), core.Namespace(rootsyncReqNamespace))
	reqNamespacedName := namespacedName(rootsyncName, rootsyncReqNamespace)
	fakeClient := syncerFake.NewClient(t, s, rsResource, secret(t, rootsyncSSHKey, secretAuth, core.Namespace(rootsyncReqNamespace)))
	testReconciler := NewRootSyncReconciler(
		fakeClient,
		controllerruntime.Log.WithName("controllers").WithName("RootSync"),
		s,
	)

	// Test creating Configmaps and Deployment resources.
	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error, got error: %q, want error: nil", err)
	}

	wantConfigMap := []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(gitSync),
			gitSyncData(branch, rootsyncRepo),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(importer),
			importerData(rootsyncDir),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantDeployment := []*appsv1.Deployment{
		rsDeploymentWithEnvFrom(
			rootsyncReqNamespace,
			"root-reconciler",
			importerDeploymentWithConfigMap(),
			rsDeploymentAnnotation(),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("diff %s", diff)
		}
	}

	// Compare ConfigMapRef field in containers.
	cmpDeployment(t, wantDeployment, fakeClient)
	t.Log("ConfigMap and Deployement successfully created")

	// Test updating Configmaps and Deployment resources.
	rsResource.Spec.Git.Revision = updatedBranch
	if err := fakeClient.Update(context.Background(), rsResource); err != nil {
		t.Fatalf("failed to update the repo sync request, got error: %v", err)
	}

	if _, err := testReconciler.Reconcile(reqNamespacedName); err != nil {
		t.Fatalf("unexpected reconciliation error upon request update, got error: %q, want error: nil", err)
	}

	wantConfigMap = []*corev1.ConfigMap{
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(gitSync),
			gitSyncData(updatedBranch, rootsyncRepo),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(importer),
			importerData(rootsyncDir),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
		configMapWithData(
			rootsyncReqNamespace,
			buildRootSyncName(SourceFormat),
			sourceFormatData(""),
			core.OwnerReference(ownerReference(rootsyncKind, rootsyncName, "")),
		),
	}

	wantDeployment = []*appsv1.Deployment{
		rsDeploymentWithEnvFrom(
			v1.NSConfigManagementSystem,
			"root-reconciler",
			importerDeploymentWithConfigMap(),
			rsDeploymentUpdatedAnnotation(),
			core.OwnerReference(ownerReference(reposyncKind, reposyncName, "")),
		),
	}

	for _, cm := range wantConfigMap {
		if diff := cmp.Diff(fakeClient.Objects[core.IDOf(cm)], cm, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("diff %s", diff)
		}
	}

	// Compare ConfigMapRef field in containers.
	cmpDeployment(t, wantDeployment, fakeClient)
	t.Log("ConfigMap and Deployement successfully updated")
}
