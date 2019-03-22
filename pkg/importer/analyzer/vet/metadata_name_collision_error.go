package vet

import (
	"sort"
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast/node"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// MetadataNameCollisionErrorCode is the error code for ObjectNameCollisionError
const MetadataNameCollisionErrorCode = "1029"

func init() {
	r1 := role()
	r1.Path = cmpath.FromSlash("namespaces/foo/r1.yaml")
	r2 := role()
	r2.Path = cmpath.FromSlash("namespaces/foo/r2.yaml")
	status.Register(MetadataNameCollisionErrorCode, MetadataNameCollisionError{
		Name:       "role",
		Duplicates: []id.Resource{r1, r2},
	})
}

// MetadataNameCollisionError reports that multiple objects in the same namespace of the same Kind share a name.
type MetadataNameCollisionError struct {
	Name       string
	Duplicates []id.Resource
}

var _ id.ResourceError = &MetadataNameCollisionError{}

// Error implements error
func (e MetadataNameCollisionError) Error() string {
	var strs []string
	for _, duplicate := range e.Duplicates {
		strs = append(strs, id.PrintResource(duplicate))
	}
	sort.Strings(strs)

	return status.Format(e,
		"Resources of the same Kind MUST have unique names in the same %[1]s and their parent %[3]ss:\n\n"+
			"%[2]s",
		node.Namespace, strings.Join(strs, "\n\n"), node.AbstractNamespace)
}

// Code implements Error
func (e MetadataNameCollisionError) Code() string { return MetadataNameCollisionErrorCode }

// Resources implements ResourceError
func (e MetadataNameCollisionError) Resources() []id.Resource {
	return e.Duplicates
}
