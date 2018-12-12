package semantic

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCountValidator ensures no more than one ConfigMap exists in system/
type ConfigMapCountValidator struct {
	Objects []ast.FileObject
}

// Validate adds an error to errorBuilder if there is more than one ConfigMap in system/
func (v ConfigMapCountValidator) Validate(errorBuilder *multierror.Builder) {
	configMaps := make(map[*corev1.ConfigMap]string)

	for _, obj := range v.Objects {
		switch configMap := obj.Object.(type) {
		case *corev1.ConfigMap:
			configMaps[configMap] = obj.Source
		}
	}

	if len(configMaps) >= 2 {
		errorBuilder.Add(vet.MultipleConfigMapsError{ConfigMaps: configMaps})
	}
}
