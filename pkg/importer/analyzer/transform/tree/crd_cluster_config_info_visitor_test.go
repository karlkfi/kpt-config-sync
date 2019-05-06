package tree_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	syncertesting "github.com/google/nomos/pkg/syncer/testing"
	"github.com/google/nomos/pkg/syncer/testing/mocks"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func namedCRD(name string) *v1beta1.CustomResourceDefinition {
	crd := fake.CustomResourceDefinition("").Object.(*v1beta1.CustomResourceDefinition).DeepCopy()
	crd.Name = name
	return crd
}

func TestCRDClusterConfigInfoVisitor(t *testing.T) {
	testCases := []struct {
		name        string
		repoCRDs    []*v1beta1.CustomResourceDefinition
		wantCRD     *v1beta1.CustomResourceDefinition
		clusterCRDs []runtime.Object
	}{
		{
			name: "empty tree, no CRDs already synced",
		},
		{
			name:     "one CRD being added",
			repoCRDs: []*v1beta1.CustomResourceDefinition{namedCRD("in-repo")},
		},
		{
			name:        "one CRD being removed",
			wantCRD:     namedCRD("on-cluster"),
			clusterCRDs: []runtime.Object{namedCRD("on-cluster")},
		},
		{
			name:        "no CRDs being removed",
			repoCRDs:    []*v1beta1.CustomResourceDefinition{namedCRD("on-cluster")},
			clusterCRDs: []runtime.Object{namedCRD("on-cluster")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := &ast.Root{}

			fakeDecoder := mocks.NewFakeDecoder(syncertesting.ToUnstructuredList(t, syncertesting.Converter, tc.clusterCRDs...))
			crdInfo, cErr := clusterconfig.NewCRDInfo(
				fakeDecoder,
				&v1.ClusterConfig{},
				tc.repoCRDs)
			if cErr != nil {
				t.Fatalf("could not generate CRDInfo: %v", cErr)
			}

			root.Accept(tree.NewCRDClusterConfigInfoVisitor(crdInfo))
			crdInfo, err := clusterconfig.GetCRDInfo(root)
			if err != nil {
				t.Fatal(err)
			}

			m := make(map[schema.GroupKind]*v1beta1.CustomResourceDefinition)
			if tc.wantCRD != nil {
				m[schema.GroupKind{}] = tc.wantCRD
			}
			want := clusterconfig.StubbedCRDInfo(m)
			if diff := cmp.Diff(want, crdInfo, cmp.AllowUnexported(clusterconfig.CRDInfo{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
