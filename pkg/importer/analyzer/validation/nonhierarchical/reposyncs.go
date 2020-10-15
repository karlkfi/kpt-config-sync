package nonhierarchical

import (
	"strings"

	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
)

// ValidateRepoSyncs validates all RepoSyncs in a passed list of FileObjects.
var ValidateRepoSyncs = PerObjectValidator(func(o ast.FileObject) status.Error {
	return ValidateRepoSync(o.Object)
})

var (
	authSSH        = v1alpha1.GitSecretSSH
	authCookiefile = v1alpha1.GitSecretCookieFile
	authGCENode    = v1alpha1.GitSecretGCENode
	authToken      = v1alpha1.GitSecretToken
	authNone       = v1alpha1.GitSecretNone
)

// ValidateRepoSync validates the content and structure of a RepoSync for any
// obvious problems.
func ValidateRepoSync(o core.Object) status.Error {
	if o.GroupVersionKind().GroupKind() != kinds.RepoSync().GroupKind() {
		return nil
	}

	if o.GetName() != v1alpha1.RepoSyncName {
		return InvalidRepoSyncName(o)
	}
	rs, ok := o.(*v1alpha1.RepoSync)
	if !ok {
		return status.InternalErrorBuilder.
			Sprint("Did not convert to RepoSync struct").BuildWithResources(o)
	}

	// We can't connect to the git repo if we don't have the URL.
	git := rs.Spec.Git
	if git.Repo == "" {
		return MissingGitRepo(o)
	}

	// Ensure auth is a valid value.
	// Note that Auth is a case-sensitive field, so ones with arbitrary capitalization
	// will fail to apply.
	switch git.Auth {
	case authSSH, authCookiefile, authGCENode, authToken, authNone:
	default:
		return InvalidAuthType(o)
	}

	// Check that proxy isn't unnecessarily declared.
	switch git.Auth {
	case authNone, authCookiefile:
		if git.Proxy != "" {
			return NoOpProxy(o)
		}
	}

	// Check the secret ref is specified if and only if it is required.
	switch git.Auth {
	case authNone, authGCENode:
		if git.SecretRef.Name != "" {
			return IllegalSecretRef(o)
		}
	default:
		if git.SecretRef.Name == "" {
			return MissingSecretRef(o)
		}
	}

	return nil
}

// InvalidRepoSyncCode is the code for an invalid declared RepoSync.
var InvalidRepoSyncCode = "1061"

var invalidRepoSyncBuilder = status.NewErrorBuilder(InvalidRepoSyncCode)

// InvalidRepoSyncName reports that a RepoSync has the wrong name.
func InvalidRepoSyncName(o core.Object) status.Error {
	name := o.GetName()
	namespace := o.GetNamespace()
	return invalidRepoSyncBuilder.
		Sprintf("RepoSyncs must be named %q, but the RepoSync for Namespace %q is named %q",
			v1alpha1.RepoSyncName, namespace, name).
		BuildWithResources(o)
}

// MissingGitRepo reports that a RepoSync doesn't declare the git repo it is
// supposed to connect to.
func MissingGitRepo(o core.Object) status.Error {
	return invalidRepoSyncBuilder.
		Sprint("RepoSyncs must define spec.git.repo").
		BuildWithResources(o)
}

// InvalidAuthType reports that a RepoSync doesn't use one of the known auth
// methods.
func InvalidAuthType(o core.Object) status.Error {
	types := []string{authSSH, authCookiefile, authGCENode, authToken, authNone}

	return invalidRepoSyncBuilder.
		Sprintf("RepoSyncs must declare spec.git.auth to be one of [%s]",
			strings.Join(types, ",")).
		BuildWithResources(o)
}

// NoOpProxy reports that a RepoSync declares a proxy, but the declaration would
// do nothing.
func NoOpProxy(o core.Object) status.Error {
	return invalidRepoSyncBuilder.
		Sprintf("RepoSyncs which declare spec.git.proxy must declare spec.git.auth=%q or %q",
			authNone, authCookiefile).
		BuildWithResources(o)
}

// IllegalSecretRef reports that a RepoSync declares an auth mode that doesn't
// allow SecretRefs does declare a SecretRef.
func IllegalSecretRef(o core.Object) status.Error {
	return invalidRepoSyncBuilder.
		Sprintf("RepoSyncs declaring spec.git.auth = %q or %q must not declare spec.git.secretRef",
			authNone, authGCENode).
		BuildWithResources(o)
}

// MissingSecretRef reports that a RepoSync declares an auth mode that requires
// a SecretRef, but does not do so.
func MissingSecretRef(o core.Object) status.Error {
	return invalidRepoSyncBuilder.
		Sprintf("RepoSyncs declaring spec.git.auth = %q or %q or %q must declare spec.git.secretRef",
			authSSH, authCookiefile, authToken).
		BuildWithResources(o)
}
