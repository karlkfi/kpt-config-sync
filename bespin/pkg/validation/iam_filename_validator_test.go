package validation

import (
	"fmt"
	"testing"

	"github.com/google/nomos/bespin/pkg/kinds"
	nomoskinds "github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/policyimporter/analyzer/vet"
	vt "github.com/google/nomos/pkg/policyimporter/analyzer/visitor/testing"
)

func TestIAMFileNameValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewIAMFilenameValidator,
		ErrorCode: vet.UndocumentedErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name: "IAMPolicy with correct fileName",
				Object: asttesting.NewFakeFileObject(
					kinds.IAMPolicy().WithVersion(""),
					fmt.Sprintf("hierarchy/foo/bar/%s", allowedIAMFilename)),
			},
			{
				Name: "IAMPolicy with incorrect fileName",
				Object: asttesting.NewFakeFileObject(
					kinds.IAMPolicy().WithVersion(""),
					fmt.Sprintf("hierarchy/foo/bar/%s", "not-allowed-Name.yaml")),
				ShouldFail: true,
			},
			{
				Name: "non-IAMPolicy with incorrect fileName",
				Object: asttesting.NewFakeFileObject(
					nomoskinds.Role(),
					fmt.Sprintf("hierarchy/foo/bar/%s", "not-allowed-Name.yaml")),
			},
		},
	}

	test.Run(t)
}
