package parse

import (
	"context"

	"github.com/google/nomos/pkg/status"
)

// Runnable represents a parser that can be pointed at and continuously parse
// a git repository.
type Runnable interface {
	// Run tells the parse.Runnable to regularly call Read and Parse until the
	// context is cancelled.
	Run(ctx context.Context)

	// Read returns the current state of the Git repository including commit hash
	// and files observed.
	Read(ctx context.Context) (*gitState, status.MultiError)

	// Parse validates and parses the current state of the Git repository into a
	// set of FileObjects.
	Parse(ctx context.Context, state *gitState) status.MultiError
}
