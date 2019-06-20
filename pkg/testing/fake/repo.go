package fake

import (
	nomosv1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/util/repo"
)

// RepoMutator modifies a Repo.
type RepoMutator func(repo *nomosv1.Repo)

// RepoMeta modifies the metadata of a Repo.
func RepoMeta(opts ...object.MetaMutator) RepoMutator {
	return func(repo *nomosv1.Repo) {
		mutate(repo, opts...)
	}
}

// RepoVersion sets the Spec.Version of a Repo.
func RepoVersion(version string) RepoMutator {
	return func(f *nomosv1.Repo) {
		f.Spec.Version = version
	}
}

// RepoObject returns an initialized Repo.
func RepoObject(opts ...RepoMutator) *nomosv1.Repo {
	result := &nomosv1.Repo{TypeMeta: toTypeMeta(kinds.Repo())}
	defaultMutate(result)
	mutate(result, object.Name("repo"))
	RepoVersion(repo.CurrentVersion)(result)
	for _, opt := range opts {
		opt(result)
	}

	return result
}

// Repo returns a default Repo with sensible defaults.
func Repo(opts ...RepoMutator) ast.FileObject {
	return fileObject(RepoObject(opts...), "system/repo.yaml")
}
