package metadata

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalLabelDefinitionErrorCode is the error code for IllegalLabelDefinitionError
const IllegalLabelDefinitionErrorCode = "1011"

var illegalLabelDefinitionError = status.NewErrorBuilder(IllegalLabelDefinitionErrorCode)

// IllegalLabelDefinitionError represent a set of illegal label definitions.
func IllegalLabelDefinitionError(resource client.Object, labels []string) status.Error {
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
