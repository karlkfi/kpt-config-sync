package semantic

import (
	"testing"

	"github.com/google/nomos/pkg/util/clusterconfig"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRDInfo adds an CRDInfo to the AST.
func CRDInfo(crdInfo *clusterconfig.CRDInfo) ast.BuildOpt {
	return func(root *ast.Root) status.MultiError {
		if crdInfo == nil {
			return nil
		}
		clusterconfig.AddCRDInfo(root, crdInfo)
		return nil
	}
}

func TestKnownResourceValidatorWithoutPendingRemovals(t *testing.T) {
	test := asttest.Validator(NewCRDRemovalValidator,
		vet.UnsupportedCRDRemovalErrorCode,
		asttest.Pass("no CRD pending delete for corresponding namespace-scoped Custom Resource",
			fake.CustomResourceDefinition(),
			fake.AnvilAtPath("namespaces/anvil.yaml"),
		),
		asttest.Pass("no CRD pending delete for corresponding cluster-scoped Custom Resource",
			fake.CustomResourceDefinition(),
			fake.AnvilAtPath("cluster/anvil.yaml"),
		),
	).With(CRDInfo(clusterconfig.StubbedCRDInfo(nil)))

	test.RunAll(t)
}

func TestKnownResourceValidatorWithPendingRemovals(t *testing.T) {
	crd := fake.CustomResourceDefinition().Object.(*v1beta1.CustomResourceDefinition)
	anvilCRDInfo := clusterconfig.StubbedCRDInfo(map[schema.GroupKind]*v1beta1.CustomResourceDefinition{
		kinds.Anvil().GroupKind(): crd,
	})

	test := asttest.Validator(NewCRDRemovalValidator,
		vet.UnsupportedCRDRemovalErrorCode,
		asttest.Pass("CRD pending delete, but no corresponding Custom Resource"),
		asttest.Fail("CRD pending delete for corresponding namespace-scoped Custom Resource",
			fake.AnvilAtPath("namespaces/anvil.yaml"),
		),
		asttest.Fail("CRD pending delete for corresponding cluster-scoped Custom Resource",
			fake.AnvilAtPath("cluster/anvil.yaml"),
		),
	).With(CRDInfo(anvilCRDInfo))

	test.RunAll(t)
}
