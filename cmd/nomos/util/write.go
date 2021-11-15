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
	// NotRunningMsg indicates that ACM is installed but not running on a cluster.
	NotRunningMsg = "NOT RUNNING"
	// NotConfiguredMsg indicates that ACM is installed but not configured for a cluster.
	NotConfiguredMsg = "NOT CONFIGURED"
	// UnknownMsg indicates that a field's value is unknown or unavailable.
	UnknownMsg = "UNKNOWN"
)

// NewWriter returns a standardized writer for the CLI for writing tabular output to the console.
func NewWriter(out io.Writer) *tabwriter.Writer {
	padding := 3
	return tabwriter.NewWriter(out, 0, 0, padding, ' ', 0)
}
