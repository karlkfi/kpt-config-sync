package veterrors

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
)

// MultipleRepoDefinitionsError reports that the system/ directory contains multiple Repo declarations.
type MultipleRepoDefinitionsError struct {
	Repos map[*v1alpha1.Repo]string
}

// Error implements error
func (e MultipleRepoDefinitionsError) Error() string {
	var repos []string
	// Sort repos so that output is deterministic.
	for r, source := range e.Repos {
		repos = append(repos, fmt.Sprintf("source: %[1]s\n"+
			"name: %[2]s", source, r.Name))
	}
	sort.Strings(repos)

	return format(e,
		"There MUST NOT be more than one Repo declaration in %[1]s/\n\n"+
			"%[2]s",
		repo.SystemDir, strings.Join(repos, "\n\n"))
}

// Code implements Error
func (e MultipleRepoDefinitionsError) Code() string { return MultipleRepoDefinitionsErrorCode }
