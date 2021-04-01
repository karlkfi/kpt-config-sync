package nonhierarchical

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func rsAuth(authType string) func(*v1alpha1.RootSync) {
	return func(sync *v1alpha1.RootSync) {
		sync.Spec.Auth = authType
	}
}

func rsName(name string) func(*v1alpha1.RootSync) {
	return func(sync *v1alpha1.RootSync) {
		sync.Name = name
	}
}

func rsProxy(proxy string) func(*v1alpha1.RootSync) {
	return func(sync *v1alpha1.RootSync) {
		sync.Spec.Proxy = proxy
	}
}

func rsSecret(secretName string) func(*v1alpha1.RootSync) {
	return func(sync *v1alpha1.RootSync) {
		sync.Spec.SecretRef.Name = secretName
	}
}

func rsGCPSAEmail(email string) func(sync *v1alpha1.RootSync) {
	return func(sync *v1alpha1.RootSync) {
		sync.Spec.GCPServiceAccountEmail = email
	}
}

func missingRootSyncRepo(rs *v1alpha1.RootSync) {
	rs.Spec.Repo = ""
}

func rootSync(opts ...func(*v1alpha1.RootSync)) *v1alpha1.RootSync {
	rs := fake.RootSyncObject()
	rs.Spec.Git.Repo = "fake repo"
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func TestValidateRootSync(t *testing.T) {
	testCases := []struct {
		name    string
		obj     *v1alpha1.RootSync
		wantErr status.Error
	}{
		{
			name: "valid",
			obj:  rootSync(rsAuth(authNone)),
		},
		{
			name:    "wrong name",
			obj:     rootSync(rsAuth(authNone), rsName("wrong name")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "missing repo",
			obj:     rootSync(rsAuth(authNone), missingRootSyncRepo),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid auth type",
			obj:     rootSync(rsAuth("invalid auth")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "no op proxy",
			obj:     rootSync(rsAuth(authNone), rsProxy("no-op proxy")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name: "valid proxy",
			obj:  rootSync(rsAuth(authGCENode), rsProxy("ok proxy")),
		},
		{
			name:    "illegal secret",
			obj:     rootSync(rsAuth(authNone), rsSecret("illegal secret")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "missing secret",
			obj:     rootSync(rsAuth(authSSH)),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid GCP serviceaccount email",
			obj:     rootSync(rsAuth(authGCPServiceAccount), rsGCPSAEmail("invalid_gcp_sa@gserviceaccount.com")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid GCP serviceaccount email with correct suffix",
			obj:     rootSync(rsAuth(authGCPServiceAccount), rsGCPSAEmail("foo@my-project.iam.gserviceaccount.com")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid GCP serviceaccount email without domain",
			obj:     rootSync(rsAuth(authGCPServiceAccount), rsGCPSAEmail("my-project")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "missing GCP serviceaccount email",
			obj:     rootSync(rsAuth(authGCPServiceAccount)),
			wantErr: fake.Error(InvalidSyncCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRootSync(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got RootSyncObject() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
