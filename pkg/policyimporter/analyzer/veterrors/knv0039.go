package veterrors

import (
	"strings"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/id"
)

// UnknownResourceVersionInSyncErrorCode is the error code for UnknownResourceVersionInSyncError
const UnknownResourceVersionInSyncErrorCode = "1039"

var unknownResourceVersionInSyncErrorExamples = []Error{
	UnknownResourceVersionInSyncError{
		Sync: &syncID{
			source:           "system/rq-sync.yaml",
			groupVersionKind: kinds.ResourceQuota().GroupKind().WithVersion("v2"),
		},
	},
}

var unknownResourceVersionInSyncErrorExplanation = `
Sample Error Message:

{{.CodeMode}}
{{index .Examples 0}}
{{.CodeMode}}
`

func init() {
	register(UnknownResourceVersionInSyncErrorCode, unknownResourceVersionInSyncErrorExamples, unknownResourceVersionInSyncErrorExplanation)
}

// UnknownResourceVersionInSyncError reports that a Sync contains a Group/Kind with an incorrect
// Version.
type UnknownResourceVersionInSyncError struct {
	id.Sync
	Allowed []string
}

// Error implements error
func (e UnknownResourceVersionInSyncError) Error() string {
	return format(e,
		"Sync defines a Resource Kind with an incorrect Version. "+
			"Known Versions: [%[1]s]\n\n"+
			"%[2]s",
		strings.Join(e.Allowed, ", "), id.PrintSync(e))
}

// Code implements Error
func (e UnknownResourceVersionInSyncError) Code() string { return UnknownResourceVersionInSyncErrorCode }
