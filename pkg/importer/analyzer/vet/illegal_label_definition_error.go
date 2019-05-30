package vet

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalLabelDefinitionErrorCode is the error code for IllegalLabelDefinitionError
const IllegalLabelDefinitionErrorCode = "1011"

func init() {
	status.AddExamples(IllegalLabelDefinitionErrorCode, IllegalLabelDefinitionError(
		role(),
		[]string{v1.SyncTokenAnnotationKey, v1.ResourceManagementKey},
	))
}

var illegalLabelDefinitionError = status.NewErrorBuilder(IllegalLabelDefinitionErrorCode)

// IllegalLabelDefinitionError represent a set of illegal label definitions.
func IllegalLabelDefinitionError(resource id.Resource, labels []string) status.Error {
	sort.Strings(labels) // ensure deterministic label order
	labels2 := make([]string, len(labels))
	for i, label := range labels {
		labels2[i] = fmt.Sprintf("%q", label)
	}
	l := strings.Join(labels2, ", ")
	return illegalLabelDefinitionError.WithResources(resource).Errorf(
		"Resources MUST NOT declare labels starting with %q. "+
			"Below Resource declares these offending labels: %s",
		v1.ConfigManagementPrefix, l)
}
