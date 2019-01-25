package vet

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/id"
)

// DuplicateSyncGroupKindErrorCode is the error code for DuplicateSyncGroupKindError
const DuplicateSyncGroupKindErrorCode = "1022"

func init() {
	register(DuplicateSyncGroupKindErrorCode, nil, "")
}

// DuplicateSyncGroupKindError reports that multiple versions were declared for the same synced kind
type DuplicateSyncGroupKindError struct {
	// Duplicates is the Group/Kind pair with duplicate definitions in Syncs.
	Duplicates []id.Sync
}

// Error implements error
func (e DuplicateSyncGroupKindError) Error() string {
	var duplicates []string
	for _, duplicate := range e.Duplicates {
		duplicates = append(duplicates, id.PrintSync(duplicate))
	}
	sort.Strings(duplicates)

	return format(e,
		"A Kind for a given Group may be declared at most once:\n\n"+
			"%[1]s",
		strings.Join(duplicates, "\n\n"))
}

// Code implements Error
func (e DuplicateSyncGroupKindError) Code() string {
	return DuplicateSyncGroupKindErrorCode
}
