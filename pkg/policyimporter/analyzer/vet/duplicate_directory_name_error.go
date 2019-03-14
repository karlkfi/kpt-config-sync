package vet

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	"github.com/google/nomos/pkg/status"
)

// DuplicateDirectoryNameErrorCode is the error code for DuplicateDirectoryNameError
const DuplicateDirectoryNameErrorCode = "1002"

func init() {
	register(DuplicateDirectoryNameErrorCode)
}

// DuplicateDirectoryNameError represents an illegal duplication of directory names.
type DuplicateDirectoryNameError struct {
	Duplicates []nomospath.Path
}

var _ status.PathError = &DuplicateDirectoryNameError{}

// Error implements error.
func (e DuplicateDirectoryNameError) Error() string {
	// Ensure deterministic node printing order.
	duplicates := make([]string, len(e.Duplicates))
	for i, duplicate := range e.Duplicates {
		duplicates[i] = duplicate.SlashPath()
	}
	sort.Strings(duplicates)
	return status.Format(e,
		"Directory names MUST be unique. Rename one of these directories:\n\n"+
			"%[1]s",
		strings.Join(duplicates, "\n"))
}

// Code implements Error
func (e DuplicateDirectoryNameError) Code() string { return DuplicateDirectoryNameErrorCode }

// RelativePaths implements PathError
func (e DuplicateDirectoryNameError) RelativePaths() []string {
	paths := make([]string, len(e.Duplicates))
	for i, dup := range e.Duplicates {
		paths[i] = dup.SlashPath()
	}
	return paths
}
