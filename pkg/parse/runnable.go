package parse

import (
	"context"

	"github.com/google/nomos/pkg/status"
)

// Runnable represents a parser that can be pointed at and continuously parse
// a git repository.
type Runnable interface {
	// Run tells the parse.Runnable to regularly call Parse until the context is
	// cancelled.
	Run(ctx context.Context)

	// Parse reads the current state of the git repository into a set of
	// FileObjects, returning an error if it encounters any problems doing so.
	Parse(ctx context.Context) status.MultiError
}
