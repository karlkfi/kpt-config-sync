package mutate

import (
	"fmt"

	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AppliedConfiguration is the annotation which is a JSON representation of the resource that was
// applied in Kubernetes.
const AppliedConfiguration = "kubectl.kubernetes.io/last-applied-configuration"

// Unapply extracts the applied configuration and replaces the Object with the applied configuration.
// Has no effect if the value in the applied configuration is not parseable.
func Unapply() Mutator {
	return func(object *ast.FileObject) {
		if applied := object.MetaObject().GetAnnotations()[AppliedConfiguration]; applied != "" {
			obj, _, err := unstructured.UnstructuredJSONScheme.Decode([]byte(applied), nil, nil)
			if err == nil {
				object.Object = obj
			} else {
				// Don't block on invalid applied annotation, just show a message to the user and continue.
				// TODO: Write to infoOut once that is approved.
				fmt.Println(err)
			}
		}
	}
}
