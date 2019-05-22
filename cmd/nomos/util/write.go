package util

import (
	"io"
	"text/tabwriter"
)

// Status enums shared across nomos subcommands.
const (
	// ErrorMsg indicates that an error has occurred when retrieving or calculating a field value.
	ErrorMsg = "ERROR"
	// NotInstalledMsg indicates that ACM is not installed on a cluster.
	NotInstalledMsg = "NOT INSTALLED"
	// UnknownMsg indicates that a field's value is unknown or unavailable.
	UnknownMsg = "UNKNOWN"
)

// NewWriter returns a standardized writer for the CLI for writing tabular output to the console.
func NewWriter(out io.Writer) *tabwriter.Writer {
	padding := 3
	minWidth := len(NotInstalledMsg) + padding
	return tabwriter.NewWriter(out, minWidth, 0, padding, ' ', 0)
}
