package validate

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// RepoSync checks if the given FileObject is a RepoSync and if so, verifies
// that its fields are valid.
func RepoSync(obj ast.FileObject) status.Error {
	if obj.GetObjectKind().GroupVersionKind().GroupKind() != kinds.RepoSync().GroupKind() {
		return nil
	}
	s, err := obj.Structured()
	if err != nil {
		return err
	}
	return RepoSyncObject(s.(*v1alpha1.RepoSync))
}

var (
	authSSH        = v1alpha1.GitSecretSSH
	authCookiefile = v1alpha1.GitSecretCookieFile
	authGCENode    = v1alpha1.GitSecretGCENode
	authToken      = v1alpha1.GitSecretToken
	authNone       = v1alpha1.GitSecretNone
)

// RepoSyncObject validates the content and structure of a RepoSync for any
// obvious problems.
func RepoSyncObject(rs *v1alpha1.RepoSync) status.Error {
	if rs.GetName() != v1alpha1.RepoSyncName {
		return nonhierarchical.InvalidRepoSyncName(rs)
	}

	// We can't connect to the git repo if we don't have the URL.
	git := rs.Spec.Git
	if git.Repo == "" {
		return nonhierarchical.MissingGitRepo(rs)
	}

	// Ensure auth is a valid value.
	// Note that Auth is a case-sensitive field, so ones with arbitrary capitalization
	// will fail to apply.
	switch git.Auth {
	case authSSH, authCookiefile, authGCENode, authToken, authNone:
	default:
		return nonhierarchical.InvalidAuthType(rs)
	}

	// Check that proxy isn't unnecessarily declared.
	switch git.Auth {
	case authNone, authCookiefile:
		if git.Proxy != "" {
			return nonhierarchical.NoOpProxy(rs)
		}
	}

	// Check the secret ref is specified if and only if it is required.
	switch git.Auth {
	case authNone, authGCENode:
		if git.SecretRef.Name != "" {
			return nonhierarchical.IllegalSecretRef(rs)
		}
	default:
		if git.SecretRef.Name == "" {
			return nonhierarchical.MissingSecretRef(rs)
		}
	}

	return nil
}
