package vet

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/policyimporter/id"
	"github.com/google/nomos/pkg/status"
)

// IllegalLabelDefinitionErrorCode is the error code for IllegalLabelDefinitionError
const IllegalLabelDefinitionErrorCode = "1011"

func init() {
	register(IllegalLabelDefinitionErrorCode, nil, "")
}

// IllegalLabelDefinitionError represent a set of illegal label definitions.
type IllegalLabelDefinitionError struct {
	id.Resource
	Labels []string
}

// Error implements error.
func (e IllegalLabelDefinitionError) Error() string {
	labels := e.Labels
	sort.Strings(labels) // ensure deterministic label order
	labels2 := make([]string, len(labels))
	for i, label := range labels {
		labels2[i] = fmt.Sprintf("%q", label)
	}
	l := strings.Join(labels2, ", ")
	return status.Format(e,
		"Resources MUST NOT declare labels starting with %[3]q. "+
			"Below Resource declares these offending labels: %[1]s\n\n"+
			"%[2]s",
		l, id.PrintResource(e), v1.NomosPrefix)
}

// Code implements Error
func (e IllegalLabelDefinitionError) Code() string { return IllegalLabelDefinitionErrorCode }
