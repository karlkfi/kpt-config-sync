package flags

import "github.com/google/nomos/cmd/nomos/repo"

const (
	// ValidateFlag is the flag to set the Validate value
	ValidateFlag = "validate"
	// PathFlag is the flag to se the Path value
	PathFlag = "path"
)

var (
	// Validate determines whether to use a schema to validate the input
	Validate bool
	// Path says where the Nomos directory is
	Path = repo.WorkingDirectoryPath
)
