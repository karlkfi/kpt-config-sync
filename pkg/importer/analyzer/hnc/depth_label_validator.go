// Package hnc adds additional HNC-understandable annotation and labels to namespaces managed by
// ACM. Please send code reviews to gke-kubernetes-hnc-core@.
package hnc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IllegalDepthLabelErrorCode is the error code for IllegalDepthLabelError.
const IllegalDepthLabelErrorCode = "1057"

var illegalDepthLabelError = status.NewErrorBuilder(IllegalDepthLabelErrorCode)

// IllegalDepthLabelError represent a set of illegal label definitions.
func IllegalDepthLabelError(resource client.Object, labels []string) status.Error {
	sort.Strings(labels) // ensure deterministic label order
	labels2 := make([]string, len(labels))
	for i, label := range labels {
		labels2[i] = fmt.Sprintf("%q", label)
	}
	l := strings.Join(labels2, ", ")
	return illegalDepthLabelError.
		Sprintf("Configs MUST NOT declare labels ending with %q. "+
			"The config has disallowed labels: %s",
			metadata.DepthSuffix, l).
		BuildWithResources(resource)
}
