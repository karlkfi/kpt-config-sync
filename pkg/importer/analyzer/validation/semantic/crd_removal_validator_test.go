package semantic

import (
	"testing"

	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/asttest"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRDInfo adds an CRDClusterConfigInfo to the AST.
func CRDInfo(crdInfo *importer.CRDClusterConfigInfo) ast.BuildOpt {
	return func(root *ast.Root) status.MultiError {
		if crdInfo == nil {
			return nil
		}
		importer.AddCRDClusterConfigInfo(root, crdInfo)
		return nil
	}
}

func TestKnownResourceValidatorWithoutPendingRemovals(t *testing.T) {
	vf := func() ast.Visitor {
		return NewCRDRemovalValidator(true)
	}

	test := asttest.Validator(vf,
		vet.UnsupportedCRDRemovalErrorCode,
		asttest.Pass("no CRD pending delete for corresponding namespace-scoped Custom Resource",
			fake.CustomResourceDefinition("cluster/crd.yaml"),
			fake.Anvil("namespaces/anvil.yaml"),
		),
		asttest.Pass("no CRD pending delete for corresponding cluster-scoped Custom Resource",
			fake.CustomResourceDefinition("cluster/crd.yaml"),
			fake.Anvil("cluster/anvil.yaml"),
		),
	).With(CRDInfo(importer.StubbedCRDClusterConfigInfo(nil)))

	test.RunAll(t)
}

func TestKnownResourceValidatorWithPendingRemovals(t *testing.T) {
	vf := func() ast.Visitor {
		return NewCRDRemovalValidator(true)
	}
	crd := fake.CustomResourceDefinition("cluster/crd.yaml").Object.(*v1beta1.CustomResourceDefinition)
	anvilCRDInfo := importer.StubbedCRDClusterConfigInfo(map[schema.GroupKind]*v1beta1.CustomResourceDefinition{
		kinds.Anvil().GroupKind(): crd,
	})

	test := asttest.Validator(vf,
		vet.UnsupportedCRDRemovalErrorCode,
		asttest.Pass("CRD pending delete, but no corresponding Custom Resource"),
		asttest.Fail("CRD pending delete for corresponding namespace-scoped Custom Resource",
			fake.Anvil("namespaces/anvil.yaml"),
		),
		asttest.Fail("CRD pending delete for corresponding cluster-scoped Custom Resource",
			fake.Anvil("cluster/anvil.yaml"),
		),
	).With(CRDInfo(anvilCRDInfo))

	test.RunAll(t)
}
