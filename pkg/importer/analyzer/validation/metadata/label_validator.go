package metadata

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/visitor"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// NewLabelValidator validates the labels declared in metadata
func NewLabelValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			var errors []string
			for l := range o.GetLabels() {
				if hasConfigManagementPrefix(l) {
					errors = append(errors, l)
				}
			}
			if errors != nil {
				return IllegalLabelDefinitionError(&o, errors)
			}
			return nil
		})
}

// IllegalLabelDefinitionErrorCode is the error code for IllegalLabelDefinitionError
const IllegalLabelDefinitionErrorCode = "1011"

var illegalLabelDefinitionError = status.NewErrorBuilder(IllegalLabelDefinitionErrorCode)

// IllegalLabelDefinitionError represent a set of illegal label definitions.
func IllegalLabelDefinitionError(resource id.Resource, labels []string) status.Error {
	sort.Strings(labels) // ensure deterministic label order
	labels2 := make([]string, len(labels))
	for i, label := range labels {
		labels2[i] = fmt.Sprintf("%q", label)
	}
	l := strings.Join(labels2, ", ")
	return illegalLabelDefinitionError.
		Sprintf("Configs MUST NOT declare labels starting with %q. "+
			"The config has disallowed labels: %s",
			v1.ConfigManagementPrefix, l).
		BuildWithResources(resource)
}
