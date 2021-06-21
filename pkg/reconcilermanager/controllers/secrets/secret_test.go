package secrets

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	syncerFake "github.com/google/nomos/pkg/syncer/syncertest/fake"
	corev1 "k8s.io/api/core/v1"
)

const (
	sshAuth        = "ssh"
	namespaceKey   = "ssh-key"
	keyData        = "test-key"
	updatedKeyData = "updated-test-key"
	ns             = "bookinfo"
)

func reposync(auth string, opts ...core.MetaMutator) *v1alpha1.RepoSync {
	result := fake.RepoSyncObject(opts...)
	result.Spec.Git = v1alpha1.Git{
		Auth:      auth,
		SecretRef: v1alpha1.SecretReference{Name: "ssh-key"},
	}
	return result
}

func secret(t *testing.T, name, data, auth string, opts ...core.MetaMutator) *corev1.Secret {
	t.Helper()
	result := fake.SecretObject(name, opts...)
	result.Data = secretData(t, data, auth)
	return result
}

func secretData(t *testing.T, data, auth string) map[string][]byte {
	t.Helper()
	key, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal test key: %v", err)
	}
	return map[string][]byte{
		auth: key,
	}
}

func fakeClient(t *testing.T, objs ...client.Object) *syncerFake.Client {
	t.Helper()
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return syncerFake.NewClient(t, s, objs...)
}

func TestCreate(t *testing.T) {
	testCases := []struct {
		name       string
		reposync   *v1alpha1.RepoSync
		client     *syncerFake.Client
		wantError  bool
		wantSecret *corev1.Secret
	}{
		{
			name:     "Secret created",
			reposync: reposync(sshAuth, core.Namespace(ns)),
			client:   fakeClient(t, secret(t, namespaceKey, keyData, sshAuth, core.Namespace(ns))),
			wantSecret: secret(t, NamespaceReconcilerSecretName(ns, namespaceKey), keyData, sshAuth,
				core.Namespace(v1.NSConfigManagementSystem),
				core.Annotation(NSReconcilerNSAnnotationKey, ns),
			),
		},
		{
			name:     "Secret updated",
			reposync: reposync(sshAuth, core.Namespace(ns)),
			client: fakeClient(t, secret(t, namespaceKey, updatedKeyData, sshAuth, core.Namespace(ns)),
				secret(t, NamespaceReconcilerSecretName(ns, namespaceKey), keyData, sshAuth, core.Namespace(v1.NSConfigManagementSystem)),
			),
			wantSecret: secret(t, NamespaceReconcilerSecretName(ns, namespaceKey), updatedKeyData, sshAuth,
				core.Namespace(v1.NSConfigManagementSystem),
				core.Annotation(NSReconcilerNSAnnotationKey, ns),
			),
		},
		{
			name:      "Secret not found",
			reposync:  reposync(sshAuth, core.Namespace(ns)),
			client:    fakeClient(t),
			wantError: true,
		},
		{
			name:      "Secret not updated, secret not present",
			reposync:  reposync(sshAuth, core.Namespace(ns)),
			client:    fakeClient(t, secret(t, NamespaceReconcilerSecretName(ns, namespaceKey), keyData, sshAuth, core.Namespace(v1.NSConfigManagementSystem))),
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Put(context.Background(), tc.reposync, tc.client)
			if tc.wantError && err == nil {
				t.Errorf("Create() got error: %q, want error", err)
			} else if !tc.wantError && err != nil {
				t.Errorf("Create() got error: %q, want error: nil", err)
			}
			if !tc.wantError {
				if diff := cmp.Diff(tc.client.Objects[core.IDOf(tc.wantSecret)], tc.wantSecret); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}
