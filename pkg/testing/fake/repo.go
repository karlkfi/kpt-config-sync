package fake

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/util/repo"
)

// RepoVersion sets the Spec.Version of a Repo.
func RepoVersion(version string) core.MetaMutator {
	return func(f core.Object) {
		f.(*v1.Repo).Spec.Version = version
	}
}

// RepoObject returns an initialized Repo.
func RepoObject(opts ...core.MetaMutator) *v1.Repo {
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
func Repo(opts ...core.MetaMutator) ast.FileObject {
	return RepoAtPath("system/repo.yaml", opts...)
}

// RepoAtPath returns a Repo at a specified path.
func RepoAtPath(path string, opts ...core.MetaMutator) ast.FileObject {
	return FileObject(RepoObject(opts...), path)
}
