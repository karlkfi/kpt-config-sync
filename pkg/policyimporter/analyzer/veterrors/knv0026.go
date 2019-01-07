package veterrors

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1/repo"
	"k8s.io/api/core/v1"
)

// MultipleConfigMapsErrorCode is the error code for MultipleConfigMapsError
const MultipleConfigMapsErrorCode = "1026"

func init() {
	register(MultipleConfigMapsErrorCode, nil, "")
}

// MultipleConfigMapsError reports that system/ declares multiple ConfigMaps.
type MultipleConfigMapsError struct {
	ConfigMaps map[*v1.ConfigMap]string
}

// Error implements error
func (e MultipleConfigMapsError) Error() string {
	var configMaps []string
	// Sort repos so that output is deterministic.
	for c, source := range e.ConfigMaps {
		configMaps = append(configMaps, fmt.Sprintf("source: %[1]s\n"+
			"name: %[2]s", source, c.Name))
	}
	sort.Strings(configMaps)

	return format(e,
		"There MUST NOT be more than one ConfigMap declaration in %[1]s/\n\n"+
			"%[2]s",
		repo.SystemDir, strings.Join(configMaps, "\n\n"))
}

// Code implements Error
func (e MultipleConfigMapsError) Code() string { return MultipleConfigMapsErrorCode }
