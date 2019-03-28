package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/testing/fake"
)

func TestDisallowedFieldsValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewDisallowedFieldsValidator,
		ErrorCode: vet.IllegalFieldsInConfigErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:   "deployment without ownerReference",
				Object: fake.Deployment("namespaces/foo/deployment.yaml"),
			},
			{
				Name:       "replicaSet with ownerReference",
				Object:     fake.Build(kinds.ReplicaSet(), object.OwnerReference("some_deployment", "some_uid", kinds.Deployment())),
				ShouldFail: true,
			},
		},
	}

	test.RunAll(t)
}
