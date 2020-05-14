package cluster

import (
	"context"

	"github.com/google/nomos/pkg/core"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client wraps a client.Client and adds convenience functions for use in
// testing.
type Client struct {
	client.Client
}

// Validate returns an error if the indicated object does not exist.
//
// TODO(b/156643819): Support predicates per go/operation-goethe-test-patterns.
func (c *Client) Validate(ctx context.Context, key client.ObjectKey, o core.Object) error {
	err := c.Get(ctx, key, o)
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
func (c *Client) ValidateNotFound(ctx context.Context, key client.ObjectKey, o core.Object) error {
	err := c.Get(ctx, key, o)
	if err == nil {
		return errors.Errorf("%T %v found", o, key)
	}
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
