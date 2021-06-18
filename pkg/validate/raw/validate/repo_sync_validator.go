package validate

import (
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/yaml"
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
	var rs *v1beta1.RepoSync
	if obj.GroupVersionKind() == kinds.RepoSync() {
		rs, err = toV1Beta1(s.(*v1alpha1.RepoSync))
		if err != nil {
			return err
		}
	} else {
		rs = s.(*v1beta1.RepoSync)
	}
	return RepoSyncObject(rs)
}

var (
	authSSH               = constants.GitSecretSSH
	authCookiefile        = constants.GitSecretCookieFile
	authGCENode           = constants.GitSecretGCENode
	authToken             = constants.GitSecretToken
	authNone              = constants.GitSecretNone
	authGCPServiceAccount = constants.GitSecretGCPServiceAccount
)

// RepoSyncObject validates the content and structure of a RepoSync for any
// obvious problems.
func RepoSyncObject(rs *v1beta1.RepoSync) status.Error {
	if rs.GetName() != constants.RepoSyncName {
		return nonhierarchical.InvalidSyncName(rs.Name, rs)
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
	case authGCPServiceAccount:
		if git.GCPServiceAccountEmail == "" {
			return nonhierarchical.MissingGCPSAEmail(rs)
		}
		if !nonhierarchical.ValidateGCPServiceAccountEmail(git.GCPServiceAccountEmail) {
			return nonhierarchical.InvalidGCPSAEmail(rs)
		}
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
	case authNone, authGCENode, authGCPServiceAccount:
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

func toV1Beta1(rs *v1alpha1.RepoSync) (*v1beta1.RepoSync, status.Error) {
	data, err := yaml.Marshal(rs)
	if err != nil {
		return nil, status.ResourceWrap(err, "failed marshalling", rs)
	}
	s := &v1beta1.RepoSync{}
	if err := yaml.Unmarshal(data, s); err != nil {
		return nil, status.ResourceWrap(err, "failed to convert to v1beta1 RepoSync", rs)
	}
	return s, nil
}
