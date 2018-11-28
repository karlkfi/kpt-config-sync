package validation

import (
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/util/multierror"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions/resource"
)

// CheckAnnotationsAndLabels verifies that the passed object has no invalid annotations or labels.
// If any exist, adds an error to the passed errorBuilder.
func CheckAnnotationsAndLabels(info *resource.Info, source string, errorBuilder *multierror.Builder) {
	object := cmdutil.AsDefaultVersionedOrOriginal(info.Object, info.Mapping)
	fileObject := ast.FileObject{Object: object, Source: source}

	checkAnnotations(fileObject, errorBuilder)
	checkLabels(fileObject, errorBuilder)
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

func checkAnnotations(o ast.FileObject, errorBuilder *multierror.Builder) {
	found := invalids(o.ToMeta().GetAnnotations(), v1alpha1.InputAnnotations)
	if len(found) != 0 {
		errorBuilder.Add(IllegalAnnotationDefinitionError{o, found})
	}
}

var noneAllowed = map[string]struct{}{}

func checkLabels(o ast.FileObject, errorBuilder *multierror.Builder) {
	found := invalids(o.ToMeta().GetLabels(), noneAllowed)
	if len(found) != 0 {
		errorBuilder.Add(IllegalLabelDefinitionError{o, found})
	}
}
