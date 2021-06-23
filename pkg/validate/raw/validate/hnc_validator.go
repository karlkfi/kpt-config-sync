package validate

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	oldhnc "github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/status"
)

// HasDepthSuffix returns true if the string ends with ".tree.hnc.x-k8s.io/depth".
func HasDepthSuffix(s string) bool {
	return strings.HasSuffix(s, oldhnc.DepthSuffix)
}

// HNCLabels verifies that the given object does not have any HNC depth labels.
func HNCLabels(obj ast.FileObject) status.Error {
	var errors []string
	for l := range obj.GetLabels() {
		if HasDepthSuffix(l) {
			errors = append(errors, l)
		}
	}
	if errors != nil {
		return oldhnc.IllegalDepthLabelError(&obj, errors)
	}
	return nil
}
