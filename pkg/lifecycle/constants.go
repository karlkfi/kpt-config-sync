// Package lifecycle defines the client-side lifecycle directives ACM honors.
//
// Implementation conforms with:
// go/lifecycle-directives-in-detail
package lifecycle

const prefix = "client.lifecycle.config.k8s.io"

// Deletion is the directive that specifies what happens when an object is
// removed from the repository.
const Deletion = prefix + "/deletion"

// PreventDeletion specifies that the resource should NOT be removed from the
// cluster if its manifest is removed from the repository.
const PreventDeletion = "prevent"
