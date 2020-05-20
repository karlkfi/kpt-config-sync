// Package hnc adds additional HNC-understandable annotation and labels to namespaces managed by
// ACM. Please send code reviews to gke-kubernetes-hnc-core@.
package hnc

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

// hasDepthSuffix returns true if the string ends with ".tree.hnc.x-k8s.io/depth".
func hasDepthSuffix(s string) bool {
	return strings.HasSuffix(s, v1.HierarchyControllerDepthSuffix)
}

// NewDepthLabelValidator ensures there's no depth label declared in metadata, which
// means there's no label ended with ".tree.hnc.x-k8s.io/depth".
func NewDepthLabelValidator() ast.Visitor {
	return visitor.NewAllObjectValidator(
		func(o ast.FileObject) status.MultiError {
			var errors []string
			for l := range o.GetLabels() {
				if hasDepthSuffix(l) {
					errors = append(errors, l)
				}
			}
			if errors != nil {
				return IllegalDepthLabelError(&o, errors)
			}
			return nil
		})
}

// IllegalDepthLabelErrorCode is the error code for IllegalDepthLabelError.
const IllegalDepthLabelErrorCode = "1057"

var illegalDepthLabelError = status.NewErrorBuilder(IllegalDepthLabelErrorCode)

// IllegalDepthLabelError represent a set of illegal label definitions.
func IllegalDepthLabelError(resource id.Resource, labels []string) status.Error {
	sort.Strings(labels) // ensure deterministic label order
	labels2 := make([]string, len(labels))
	for i, label := range labels {
		labels2[i] = fmt.Sprintf("%q", label)
	}
	l := strings.Join(labels2, ", ")
	return illegalDepthLabelError.
		Sprintf("Configs MUST NOT declare labels ending with %q. "+
			"The config has disallowed labels: %s",
			v1.HierarchyControllerDepthSuffix, l).
		BuildWithResources(resource)
}
