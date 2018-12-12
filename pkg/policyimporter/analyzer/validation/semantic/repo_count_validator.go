package semantic

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime"
)

// RepoCountValidator ensures the correct number of Repo Resources are defined in system/
type RepoCountValidator struct {
	Objects map[runtime.Object]string
}

// Validate adds an error to errorBuilder if there are an incorrect number of Repo Resources
func (v RepoCountValidator) Validate(errorBuilder *multierror.Builder) {
	repos := make(map[*v1alpha1.Repo]string)

	for obj, source := range v.Objects {
		switch repo := obj.(type) {
		case *v1alpha1.Repo:
			repos[repo] = source
		}
	}

	if len(repos) == 0 {
		errorBuilder.Add(vet.MissingRepoError{})
	} else if len(repos) >= 2 {
		errorBuilder.Add(vet.MultipleRepoDefinitionsError{Repos: repos})
	}
}
