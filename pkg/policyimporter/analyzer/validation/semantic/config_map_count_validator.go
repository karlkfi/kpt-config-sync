package semantic

import (
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	"github.com/google/nomos/pkg/util/multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConfigMapCountValidator ensures no more than one ConfigMap exists in system/
type ConfigMapCountValidator struct {
	Objects map[runtime.Object]string
}

// Validate adds an error to errorBuilder if there is more than one ConfigMap in system/
func (v ConfigMapCountValidator) Validate(errorBuilder *multierror.Builder) {
	configMaps := make(map[*corev1.ConfigMap]string)

	for obj, source := range v.Objects {
		switch configMap := obj.(type) {
		case *corev1.ConfigMap:
			configMaps[configMap] = source
		}
	}

	if len(configMaps) >= 2 {
		errorBuilder.Add(vet.MultipleConfigMapsError{ConfigMaps: configMaps})
	}
}
