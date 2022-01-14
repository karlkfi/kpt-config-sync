package controllers

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/reconciler"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestMapSecretToRootSync(t *testing.T) {
	testCases := []struct {
		name   string
		secret client.Object
		want   []reconcile.Request
	}{
		{
			name:   "A secret from the default namespace",
			secret: fake.SecretObject("s1", core.Namespace("default")),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(fmt.Sprintf("%s-bookstore", reconciler.NsReconcilerPrefix), core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace NOT starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject("s1", core.Namespace(configsync.ControllerNamespace)),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RootSyncName,
						Namespace: configsync.ControllerNamespace,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapSecretToRootSync()(tc.secret)
			if diff := cmp.Diff(tc.want, result); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestMapSecretToRepoSync(t *testing.T) {
	testCases := []struct {
		name   string
		secret client.Object
		want   []reconcile.Request
	}{
		{
			name:   "A secret from the default namespace",
			secret: fake.SecretObject("s1", core.Namespace("default")),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "default",
					},
				},
			},
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace NOT starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject("s1", core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name: fmt.Sprintf("A secret from the %s namespace starting with %s and having the %s annotation",
				configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-", NSReconcilerNSAnnotationKey),
			secret: fake.SecretObject(fmt.Sprintf("%s-gamestore-token-123-ssh-key", reconciler.NsReconcilerPrefix),
				core.Namespace(configsync.ControllerNamespace),
				core.Annotation(NSReconcilerNSAnnotationKey, "bookstore"),
			),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "bookstore",
					},
				},
			},
		},
		{
			name: fmt.Sprintf("A secret from the %s namespace starting with %s, including `-token-`, and ending with `-ssh-key`", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(fmt.Sprintf("%s-gamestore-token-123-ssh-key", reconciler.NsReconcilerPrefix),
				core.Namespace(configsync.ControllerNamespace),
			),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
		{
			name: fmt.Sprintf("A secret from the %s namespace starting with %s and ending with `-ssh-key`", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(fmt.Sprintf("%s-gamestore-1-ssh-key", reconciler.NsReconcilerPrefix),
				core.Namespace(configsync.ControllerNamespace),
			),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore-1",
					},
				},
			},
		},
		{
			name: fmt.Sprintf("A secret from the %s namespace starting with %s and including `-token-`", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(fmt.Sprintf("%s-gamestore-token-133", reconciler.NsReconcilerPrefix),
				core.Namespace(configsync.ControllerNamespace),
			),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
		{
			name: fmt.Sprintf("A secret from the %s namespace starting with %s and including neither `-token-` nor the `-ssh-key` suffix", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			secret: fake.SecretObject(fmt.Sprintf("%s-gamestore-git-creds", reconciler.NsReconcilerPrefix),
				core.Namespace(configsync.ControllerNamespace),
			),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore-git-creds",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapSecretToRepoSync()(tc.secret)
			if diff := cmp.Diff(tc.want, result); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestMapObjectToRepoSync(t *testing.T) {
	testCases := []struct {
		name   string
		object client.Object
		want   []reconcile.Request
	}{
		// ConfigMap
		{
			name:   "A configmap from the default namespace",
			object: fake.ConfigMapObject(core.Name("cm1"), core.Namespace("default")),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace NOT starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.ConfigMapObject(core.Name("cm1"), core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace starting with %s and with the `-reconciler` suffix", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.ConfigMapObject(core.Name(fmt.Sprintf("%s-gamestore-reconciler", reconciler.NsReconcilerPrefix)), core.Namespace(configsync.ControllerNamespace)),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace starting with %s and with the `-git-sync` suffix", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.ConfigMapObject(core.Name(fmt.Sprintf("%s-gamestore-git-sync", reconciler.NsReconcilerPrefix)), core.Namespace(configsync.ControllerNamespace)),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
		{
			name:   fmt.Sprintf("A secret from the %s namespace starting with %s and without the `-reconciler` and `-git-sync` suffix", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.ConfigMapObject(core.Name(fmt.Sprintf("%s-gamestore", reconciler.NsReconcilerPrefix)), core.Namespace(configsync.ControllerNamespace)),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
		// Deployment
		{
			name:   "A deployment from the default namespace",
			object: fake.DeploymentObject(core.Name("deploy1"), core.Namespace("default")),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A deployment from the %s namespace NOT starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.DeploymentObject(core.Name("deploy1"), core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A deployment from the %s namespace starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.DeploymentObject(core.Name(fmt.Sprintf("%s-gamestore", reconciler.NsReconcilerPrefix)), core.Namespace(configsync.ControllerNamespace)),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
		// ServiceAccount
		{
			name:   "A serviceaccount from the default namespace",
			object: fake.ServiceAccountObject("sa1", core.Namespace("default")),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A serviceaccount from the %s namespace NOT starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.ServiceAccountObject("sa1", core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A serviceaccount from the %s namespace starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.ServiceAccountObject(fmt.Sprintf("%s-gamestore", reconciler.NsReconcilerPrefix), core.Namespace(configsync.ControllerNamespace)),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
		// RoleBinding
		{
			name:   "A rolebinding from the default namespace",
			object: fake.RoleBindingObject(core.Name("rb1"), core.Namespace("default")),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A rolebinding from the %s namespace NOT starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.RoleBindingObject(core.Name("rb1"), core.Namespace(configsync.ControllerNamespace)),
			want:   nil,
		},
		{
			name:   fmt.Sprintf("A rolebinding from the %s namespace starting with %s", configsync.ControllerNamespace, reconciler.NsReconcilerPrefix+"-"),
			object: fake.RoleBindingObject(core.Name(fmt.Sprintf("%s-gamestore", reconciler.NsReconcilerPrefix)), core.Namespace(configsync.ControllerNamespace)),
			want: []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      configsync.RepoSyncName,
						Namespace: "gamestore",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mapObjectToRepoSync()(tc.object)
			if diff := cmp.Diff(tc.want, result); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
