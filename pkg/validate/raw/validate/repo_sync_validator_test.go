package validate

import (
	"errors"
	"testing"

	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
)

func auth(authType string) func(*v1alpha1.RepoSync) {
	return func(sync *v1alpha1.RepoSync) {
		sync.Spec.Auth = authType
	}
}

func named(name string) func(*v1alpha1.RepoSync) {
	return func(sync *v1alpha1.RepoSync) {
		sync.Name = name
	}
}

func proxy(proxy string) func(*v1alpha1.RepoSync) {
	return func(sync *v1alpha1.RepoSync) {
		sync.Spec.Proxy = proxy
	}
}

func secret(secretName string) func(*v1alpha1.RepoSync) {
	return func(sync *v1alpha1.RepoSync) {
		sync.Spec.SecretRef.Name = secretName
	}
}

func missingRepo(rs *v1alpha1.RepoSync) {
	rs.Spec.Repo = ""
}

func repoSync(opts ...func(*v1alpha1.RepoSync)) *v1alpha1.RepoSync {
	rs := fake.RepoSyncObject()
	rs.Spec.Git.Repo = "fake repo"
	for _, opt := range opts {
		opt(rs)
	}
	return rs
}

func TestRepoSyncObject(t *testing.T) {
	testCases := []struct {
		name    string
		obj     *v1alpha1.RepoSync
		wantErr status.Error
	}{
		{
			name: "valid",
			obj:  repoSync(auth(authNone)),
		},
		{
			name:    "wrong name",
			obj:     repoSync(auth(authNone), named("wrong name")),
			wantErr: fake.Error(nonhierarchical.InvalidRepoSyncCode),
		},
		{
			name:    "missing repo",
			obj:     repoSync(auth(authNone), missingRepo),
			wantErr: fake.Error(nonhierarchical.InvalidRepoSyncCode),
		},
		{
			name:    "invalid auth type",
			obj:     repoSync(auth("invalid auth")),
			wantErr: fake.Error(nonhierarchical.InvalidRepoSyncCode),
		},
		{
			name:    "no op proxy",
			obj:     repoSync(auth(authNone), proxy("no-op proxy")),
			wantErr: fake.Error(nonhierarchical.InvalidRepoSyncCode),
		},
		{
			name: "valid proxy",
			obj:  repoSync(auth(authGCENode), proxy("ok proxy")),
		},
		{
			name:    "illegal secret",
			obj:     repoSync(auth(authNone), secret("illegal secret")),
			wantErr: fake.Error(nonhierarchical.InvalidRepoSyncCode),
		},
		{
			name:    "missing secret",
			obj:     repoSync(auth(authSSH)),
			wantErr: fake.Error(nonhierarchical.InvalidRepoSyncCode),
		},
		{
			name: "valid secret",
			obj:  repoSync(auth(authSSH), secret("valid secret")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := RepoSyncObject(tc.obj)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Got RepoSyncObject() error %v, want %v", err, tc.wantErr)
			}
		})
	}
}
