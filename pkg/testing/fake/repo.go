package fake

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/util/repo"
)

// RepoMutator modifies a Repo.
type RepoMutator func(repo *v1.Repo)

// RepoVersion sets the Spec.Version of a Repo.
func RepoVersion(version string) RepoMutator {
	return func(f *v1.Repo) {
		f.Spec.Version = version
	}
}

// RepoObject returns an initialized Repo.
func RepoObject(opts ...RepoMutator) *v1.Repo {
	result := &v1.Repo{TypeMeta: toTypeMeta(kinds.Repo())}
	defaultMutate(result)
	mutate(result, core.Name("repo"))
	RepoVersion(repo.CurrentVersion)(result)
	for _, opt := range opts {
		opt(result)
	}

	return result
}

// Repo returns a default Repo with sensible defaults.
func Repo(opts ...RepoMutator) ast.FileObject {
	return RepoAtPath("system/repo.yaml", opts...)
}

// RepoAtPath returns a Repo at a specified path.
func RepoAtPath(path string, opts ...RepoMutator) ast.FileObject {
	return FileObject(RepoObject(opts...), path)
}
