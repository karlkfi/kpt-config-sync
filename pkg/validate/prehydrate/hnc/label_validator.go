package hnc

import (
	"strings"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	oldhnc "github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/parsed"
)

// hasDepthSuffix returns true if the string ends with ".tree.hnc.x-k8s.io/depth".
func hasDepthSuffix(s string) bool {
	return strings.HasSuffix(s, oldhnc.DepthSuffix)
}

// DepthLabelValidator ensures there's no depth label declared in metadata, which
// means there's no label ended with ".tree.hnc.x-k8s.io/depth".
func DepthLabelValidator() parsed.ValidatorFunc {
	return parsed.ValidateAllObjects(parsed.PerObjectVisitor(validateDepthLabels))
}

func validateDepthLabels(obj ast.FileObject) status.Error {
	var errors []string
	for l := range obj.GetLabels() {
		if hasDepthSuffix(l) {
			errors = append(errors, l)
		}
	}
	if errors != nil {
		return oldhnc.IllegalDepthLabelError(&obj, errors)
	}
	return nil
}
