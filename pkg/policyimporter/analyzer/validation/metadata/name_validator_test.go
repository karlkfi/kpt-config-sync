package metadata

import (
	"testing"

	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors"
	"github.com/google/nomos/pkg/policyimporter/analyzer/veterrors/veterrorstest"
	"github.com/google/nomos/pkg/util/multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type nameTestCase struct {
	testName     string
	resourceName string
	source       string
	gvk          schema.GroupVersionKind
	error        []string
}

var nameTestCases = []nameTestCase{
	{
		testName:     "empty name",
		resourceName: "",
		error:        []string{veterrors.MissingObjectNameErrorCode},
	},
	{
		testName:     "legal name",
		resourceName: "name",
	},
	{
		testName:     "illegal crd name",
		resourceName: "Name",
		gvk:          kinds.Repo(),
		error:        []string{veterrors.InvalidMetadataNameErrorCode},
	},
	{
		testName:     "legal crd name",
		resourceName: "name",
		gvk:          kinds.Repo(),
	},
	{
		testName:     "non crd with illegal crd name",
		gvk:          kinds.ResourceQuota(),
		resourceName: "Name",
	},
	{
		testName:     "illegal top level namespace",
		resourceName: "namespaces",
		source:       "namespaces/ns.yaml",
		gvk:          kinds.Namespace(),
		error:        []string{veterrors.IllegalTopLevelNamespaceErrorCode},
	},
	{
		testName:     "illegal namespace name",
		resourceName: "bar",
		source:       "namespaces/foo/ns.yaml",
		gvk:          kinds.Namespace(),
		error:        []string{veterrors.InvalidNamespaceNameErrorCode},
	},
	{
		testName:     "legal namespace name",
		resourceName: "foo",
		source:       "namespaces/foo/ns.yaml",
		gvk:          kinds.Namespace(),
	},
}

func (tc nameTestCase) Run(t *testing.T) {
	meta := resourceMeta{name: tc.resourceName, source: tc.source, groupVersionKind: tc.gvk}

	eb := multierror.Builder{}
	NameValidatorFactory.New([]ResourceMeta{meta}).Validate(&eb)

	veterrorstest.ExpectErrors(tc.error, eb.Build(), t)
}

func TestNameValidator(t *testing.T) {
	for _, tc := range nameTestCases {
		t.Run(tc.testName, tc.Run)
	}
}
