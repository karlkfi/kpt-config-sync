package syntax

import (
	"testing"

	"github.com/google/nomos/pkg/testing/fake"

	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/vet"
	vt "github.com/google/nomos/pkg/importer/analyzer/visitor/testing"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/kinds"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func crd(path string, gvk schema.GroupVersionKind) ast.FileObject {
	return ast.FileObject{
		Path: cmpath.FromSlash(path),
		Object: &v1beta1.CustomResourceDefinition{
			TypeMeta: v1.TypeMeta{
				APIVersion: kinds.CustomResourceDefinition().GroupVersion().String(),
				Kind:       kinds.CustomResourceDefinition().Kind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "crd",
			},
			Spec: v1beta1.CustomResourceDefinitionSpec{
				Group: gvk.Group,
				Names: v1beta1.CustomResourceDefinitionNames{
					Kind: gvk.Kind,
				},
			},
		},
	}
}

func TestDisallowedCRDsValidator(t *testing.T) {
	test := vt.ObjectValidatorTest{
		Validator: NewDisallowedCRDsValidator,
		ErrorCode: vet.UnsupportedObjectErrorCode,
		TestCases: []vt.ObjectValidatorTestCase{
			{
				Name:       "clusterconfig CRD",
				Object:     crd("cluster/clusterconfig-crd.yaml", kinds.ClusterConfig()),
				ShouldFail: true,
			},
			{
				Name:       "namespaceconfig CRD",
				Object:     crd("cluster/namespaceconfig-crd.yaml", kinds.NamespaceConfig()),
				ShouldFail: true,
			},
			{
				Name:       "sync CRD",
				Object:     crd("cluster/sync-crd.yaml", kinds.Sync()),
				ShouldFail: true,
			},
			{
				Name:   "non-anthos config management CRD",
				Object: crd("cluster/anvil-crd.yaml", kinds.Anvil()),
			},
			{
				Name:   "non-CRD config",
				Object: fake.ClusterRole("cluster/clusterrole.yaml"),
			},
		},
	}

	test.RunAll(t)
}
