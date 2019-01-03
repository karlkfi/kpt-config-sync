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
	// Path is the repository path in which the conflict happened
	Path string
	// Cluster is the cluster in which the conflict happened
	Cluster    string
	Duplicates []ResourceID
}

// Error implements error.
func (e ConflictingResourceQuotaError) Error() string {
	var strs sort.StringSlice
	for _, duplicate := range e.Duplicates {
		strs = append(strs, printResourceID(duplicate))
	}
	strs.Sort()

	if e.Cluster != "" {
		return format(e,
			"A directory MUST NOT contain more than one ResourceQuota "+
				"Resource targeted to cluster %[3]q.  "+
				"Directory %[1]q contains multiple ResourceQuota Resources:\n\n"+
				"%[2]s",
			e.Path, strings.Join(strs, "\n\n"), e.Cluster)
	}

	return format(e,
		"A directory MUST NOT contain more than one ResourceQuota Resource. "+
			"Directory %[1]q contains multiple ResourceQuota Resources:\n\n"+
			"%[2]s",
		e.Path, strings.Join(strs, "\n\n"))
}

// Code implements Error
func (e ConflictingResourceQuotaError) Code() string { return ConflictingResourceQuotaErrorCode }
