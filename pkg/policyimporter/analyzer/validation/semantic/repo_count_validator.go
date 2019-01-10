package semantic

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/util/multierror"
)

// RepoCountValidator ensures the correct number of Repo Resources are defined in system/
type RepoCountValidator struct {
	Objects []ast.FileObject
}

// Validate adds an error to errorBuilder if there are an incorrect number of Repo Resources
func (v RepoCountValidator) Validate(errorBuilder *multierror.Builder) {
	repos := make(map[*v1alpha1.Repo]nomospath.Relative)

	for _, obj := range v.Objects {
		switch repo := obj.Object.(type) {
		case *v1alpha1.Repo:
			repos[repo] = obj.Relative
		}
	}

	if len(repos) == 0 {
		errorBuilder.Add(veterrors.MissingRepoError{})
	} else if len(repos) >= 2 {
		errorBuilder.Add(veterrors.MultipleRepoDefinitionsError{Repos: repos})
	}
}
