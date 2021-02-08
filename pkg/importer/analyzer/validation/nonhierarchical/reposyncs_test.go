package nonhierarchical

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
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

func TestValidateRepoSync(t *testing.T) {
	testCases := []struct {
		name string
		obj  *v1alpha1.RepoSync
		want error
	}{
		{
			name: "valid",
			obj:  repoSync(auth(authNone)),
			want: nil,
		},
		{
			name: "wrong name",
			obj:  repoSync(auth(authNone), named("wrong name")),
			want: invalidRepoSyncBuilder.Build(),
		},
		{
			name: "missing repo",
			obj:  repoSync(auth(authNone), missingRepo),
			want: invalidRepoSyncBuilder.Build(),
		},
		{
			name: "invalid auth type",
			obj:  repoSync(auth("invalid auth")),
			want: invalidRepoSyncBuilder.Build(),
		},
		{
			name: "no op proxy",
			obj:  repoSync(auth(authNone), proxy("no-op proxy")),
			want: invalidRepoSyncBuilder.Build(),
		},
		{
			name: "valid proxy",
			obj:  repoSync(auth(authGCENode), proxy("ok proxy")),
		},
		{
			name: "illegal secret",
			obj:  repoSync(auth(authNone), secret("illegal secret")),
			want: invalidRepoSyncBuilder.Build(),
		},
		{
			name: "missing secret",
			obj:  repoSync(auth(authSSH)),
			want: invalidRepoSyncBuilder.Build(),
		},
		{
			name: "valid secret",
			obj:  repoSync(auth(authSSH), secret("valid secret")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateRepoSync(tc.obj)
			if !errors.Is(tc.want, got) {
				t.Error(cmp.Diff(tc.want, got))
			}
		})
	}
}
