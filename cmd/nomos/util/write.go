package util

import (
	"io"
	"text/tabwriter"
)

// NewWriter returns a standardized writer for the CLI for writing tabular output to the console.
func NewWriter(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, 10, 0, 3, ' ', 0)
}
