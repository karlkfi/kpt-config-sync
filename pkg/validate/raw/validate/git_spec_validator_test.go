package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func auth(authType string) func(*v1beta1.RepoSync) {
	return func(sync *v1beta1.RepoSync) {
		sync.Spec.Auth = authType
	}
}

func named(name string) func(*v1beta1.RepoSync) {
	return func(sync *v1beta1.RepoSync) {
		sync.Name = name
	}
}

func proxy(proxy string) func(*v1beta1.RepoSync) {
	return func(sync *v1beta1.RepoSync) {
		sync.Spec.Proxy = proxy
	}
}

func secret(secretName string) func(*v1beta1.RepoSync) {
	return func(sync *v1beta1.RepoSync) {
		sync.Spec.SecretRef.Name = secretName
	}
}

func gcpSAEmail(email string) func(sync *v1beta1.RepoSync) {
	return func(sync *v1beta1.RepoSync) {
		sync.Spec.GCPServiceAccountEmail = email
	}
}

func missingRepo(rs *v1beta1.RepoSync) {
	rs.Spec.Repo = ""
}

func repoSync(opts ...func(*v1beta1.RepoSync)) *v1beta1.RepoSync {
	rs := fake.RepoSyncObjectV1Beta1("test-ns", configsync.RepoSyncName)
	rs.Spec.Git.Repo = "fake repo"
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func TestValidateGitSpec(t *testing.T) {
	testCases := []struct {
		name    string
		obj     *v1beta1.RepoSync
		wantErr status.Error
	}{
		{
			name: "valid",
			obj:  repoSync(auth(authNone)),
		},
		{
			name: "a user-defined name",
			obj:  repoSync(auth(authNone), named("user-defined-repo-sync-name")),
		},
		{
			name:    "missing repo",
			obj:     repoSync(auth(authNone), missingRepo),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid auth type",
			obj:     repoSync(auth("invalid auth")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "no op proxy",
			obj:     repoSync(auth(authGCENode), proxy("no-op proxy")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name: "valid proxy with none auth type",
			obj:  repoSync(auth(authNone), proxy("ok proxy")),
		},
		{
			name: "valid proxy with cookiefile",
			obj:  repoSync(auth(authCookiefile), secret("cookiefile"), proxy("ok proxy")),
		},
		{
			name: "valid proxy with token",
			obj:  repoSync(auth(authToken), secret("token"), proxy("ok proxy")),
		},
		{
			name:    "illegal secret",
			obj:     repoSync(auth(authNone), secret("illegal secret")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "missing secret",
			obj:     repoSync(auth(authSSH)),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid GCP serviceaccount email",
			obj:     repoSync(auth(authGCPServiceAccount), gcpSAEmail("invalid_gcp_sa@gserviceaccount.com")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid GCP serviceaccount email with correct suffix",
			obj:     repoSync(auth(authGCPServiceAccount), gcpSAEmail("foo@my-project.iam.gserviceaccount.com")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "invalid GCP serviceaccount email without domain",
			obj:     repoSync(auth(authGCPServiceAccount), gcpSAEmail("my-project")),
			wantErr: fake.Error(InvalidSyncCode),
		},
		{
			name:    "missing GCP serviceaccount email",
			obj:     repoSync(auth(authGCPServiceAccount)),
			wantErr: fake.Error(InvalidSyncCode),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := GitSpec(tc.obj.Spec.Git, tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got ValidateGitSpec() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
