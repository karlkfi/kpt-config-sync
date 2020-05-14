package nomostest

import (
	"context"

	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NT represents the test environment for a single Nomos end-to-end test case.
type NT struct {
	// Name is the unique name of the test run.
	Name string

	// TmpDir is the temporary directory the test will write to.
	// By default, automatically deleted when the test finishes.
	TmpDir string

	// Client is the underlying client used to talk to the Kubernetes cluster.
	//
	// Most tests shouldn't need to talk directly to this, unless simulating
	// direct interactions with the API Server.
	client.Client
}

// Validate returns an error if the indicated object does not exist.
//
// TODO(b/156643819): Support predicates per go/operation-goethe-test-patterns.
func (nt *NT) Validate(ctx context.Context, key client.ObjectKey, o core.Object) error {
	err := nt.Get(ctx, key, o)
	if err != nil {
		return err
	}
	return nil
}

// ValidateNotFound returns an error if the indicated object exists.
//
// `o` must either be:
// 1) a struct pointer to the type of the object to search for, or
// 2) an unstructured.Unstructured with the type information filled in.
func (nt *NT) ValidateNotFound(ctx context.Context, key client.ObjectKey, o core.Object) error {
	err := nt.Get(ctx, key, o)
	if err == nil {
		return errors.Errorf("%T %v found", o, key)
	}
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
