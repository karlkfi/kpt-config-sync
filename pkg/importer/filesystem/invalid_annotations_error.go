package filesystem

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"k8s.io/api/rbac/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InvalidAnnotationValueErrorCode is the error code for when a value in
// metadata.annotations is not a string.
const InvalidAnnotationValueErrorCode = "1054"

func init() {
	o := ast.NewFileObject(
		&v1alpha1.Role{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.Role().GroupVersion().String(),
				Kind:       kinds.Role().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "role",
			},
		},
		cmpath.FromSlash("namespaces/foo/role.yaml"),
	)
	status.AddExamples(InvalidAnnotationValueErrorCode, invalidAnnotationValueError(
		&o, []string{"foo", "bar"},
	))
}

var invalidAnnotationValueErrorBase = status.NewErrorBuilder(InvalidAnnotationValueErrorCode)

// IllegalKindInClusterError reports that an object has been illegally defined in cluster/
func invalidAnnotationValueError(resource id.Resource, keys []string) status.Error {
	return invalidAnnotationValueErrorBase.WithResources(resource).Errorf(
		"Values in metadata.annotations MUST be strings. "+
			`To fix, add quotes around the values. Non-string values for:

metadata.annotations.%s `,
		strings.Join(keys, "\nmetadata.annotations."))
}
