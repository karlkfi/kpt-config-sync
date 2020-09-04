package ntopts

import (
	"github.com/google/nomos/pkg/importer/filesystem"
)

// Nomos configures options for installing Nomos on the test cluster.
type Nomos struct {
	filesystem.SourceFormat

	// MultiRepo indicates that NT should setup and test multi-repo behavior
	// rather than mono-repo behavior.
	MultiRepo bool
}
