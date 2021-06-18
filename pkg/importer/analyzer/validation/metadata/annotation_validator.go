package metadata

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/constants"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalAnnotationDefinitionErrorCode is the error code for IllegalAnnotationDefinitionError
const IllegalAnnotationDefinitionErrorCode = "1010"

var illegalAnnotationDefinitionError = status.NewErrorBuilder(IllegalAnnotationDefinitionErrorCode)

// IllegalAnnotationDefinitionError represents a set of illegal annotation definitions.
func IllegalAnnotationDefinitionError(resource client.Object, annotations []string) status.Error {
	sort.Strings(annotations) // ensure deterministic annotation order
	annotations2 := make([]string, len(annotations))
	for i, annotation := range annotations {
		annotations2[i] = fmt.Sprintf("%q", annotation)
	}
	a := strings.Join(annotations2, ", ")
	return illegalAnnotationDefinitionError.
		Sprintf("Configs MUST NOT declare unsupported annotations starting with %q or %q. "+
			"The config has invalid annotations: %s",
			v1.ConfigManagementPrefix, constants.ConfigSyncPrefix, a).
		BuildWithResources(resource)
}
