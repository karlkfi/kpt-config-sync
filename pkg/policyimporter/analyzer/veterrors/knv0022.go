package veterrors

import (
	"sort"
	"strings"
)

// DuplicateSyncGroupKindError reports that multiple versions were declared for the same synced kind
type DuplicateSyncGroupKindError struct {
	// Duplicates is the Group/Kind pair with duplicate definitions in Syncs.
	Duplicates []SyncID
}

// Error implements error
func (e DuplicateSyncGroupKindError) Error() string {
	var duplicates []string
	for _, duplicate := range e.Duplicates {
		duplicates = append(duplicates, printSyncID(duplicate))
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
