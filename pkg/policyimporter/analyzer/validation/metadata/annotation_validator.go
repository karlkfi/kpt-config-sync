package metadata

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
)

// AnnotationValidator validates the annotations in a ast.FileObject
var AnnotationValidator = &syntax.FileObjectValidator{
	ValidateFn: func(o ast.FileObject) error {
		found := invalids(o.ToMeta().GetAnnotations(), v1alpha1.InputAnnotations)
		if len(found) > 0 {
			return veterrors.IllegalAnnotationDefinitionError{ResourceID: &o, Annotations: found}
		}
		return nil
	},
}

func invalids(m map[string]string, allowed map[string]struct{}) []string {
	var found []string

	for k := range m {
		if _, found := allowed[k]; found {
			continue
		}
		if strings.HasPrefix(k, policyhierarchy.GroupName+"/") {
			found = append(found, fmt.Sprintf("%q", k))
		}
	}

	return found
}
