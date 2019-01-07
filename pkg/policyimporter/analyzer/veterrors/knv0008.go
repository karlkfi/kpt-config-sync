package veterrors

import (
	"sort"
	"strings"
)

// ConflictingResourceQuotaErrorCode is the error code for ConflictingResourceQuotaError
const ConflictingResourceQuotaErrorCode = "1008"

func init() {
	register(ConflictingResourceQuotaErrorCode, nil, "")
}

// ConflictingResourceQuotaError represents multiple ResourceQuotas illegally presiding in the same directory.
type ConflictingResourceQuotaError struct {
	Path       string
	Duplicates []ResourceID
}

// Error implements error.
func (e ConflictingResourceQuotaError) Error() string {
	var strs []string
	for _, duplicate := range e.Duplicates {
		strs = append(strs, printResourceID(duplicate))
	}
	sort.Strings(strs)

	return format(e,
		"A directory MUST NOT contain more than one ResourceQuota Resource. "+
			"Directory %[1]q contains multiple ResourceQuota Resources:\n\n"+
			"%[2]s",
		e.Path, strings.Join(strs, "\n\n"))
}

// Code implements Error
func (e ConflictingResourceQuotaError) Code() string { return ConflictingResourceQuotaErrorCode }
