package tree_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/google/go-cmp/cmp"
	"github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/tree"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestCRDClusterConfigInfoVisitor(t *testing.T) {
	crd := fake.CustomResourceDefinition("cluster/crd.yaml").Object.(*v1beta1.CustomResourceDefinition)

	testCases := []struct {
		name          string
		objects       []*ast.ClusterObject
		crd           *v1beta1.CustomResourceDefinition
		clusterConfig *v1.ClusterConfig
	}{
		{
			name:          "empty tree, no CRDs already synced",
			clusterConfig: &v1.ClusterConfig{},
		},
		{
			name: "one CRD being added",
			objects: []*ast.ClusterObject{
				{FileObject: fake.CustomResourceDefinition("cluster/crd.yaml")},
			},
			clusterConfig: &v1.ClusterConfig{},
		},
		{
			name: "one CRDs being removed",
			crd:  crd,
			clusterConfig: &v1.ClusterConfig{
				Spec: v1.ClusterConfigSpec{
					Resources: []v1.GenericResources{
						{
							Group: "apiextensions.k8s.io",
							Kind:  crd.Kind,
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1beta1",
									Objects: []runtime.RawExtension{
										{
											Object: crd,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no CRDs being removed",
			objects: []*ast.ClusterObject{
				{FileObject: fake.CustomResourceDefinition("cluster/crd.yaml")},
			},
			clusterConfig: &v1.ClusterConfig{
				Spec: v1.ClusterConfigSpec{
					Resources: []v1.GenericResources{
						{
							Group: "apiextensions.k8s.io",
							Kind:  crd.Kind,
							Versions: []v1.GenericVersionResources{
								{
									Version: "v1beta1",
									Objects: []runtime.RawExtension{
										{
											Object: crd,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := &ast.Root{}
			root.ClusterObjects = tc.objects
			crdInfo := importer.NewCRDClusterConfigInfo(tc.clusterConfig, root.ClusterObjects)
			root.Accept(tree.NewCRDClusterConfigInfoVisitor(crdInfo))
			crdInfo, err := importer.GetCRDClusterConfigInfo(root)
			if err != nil {
				t.Fatal(err)
			}

			m := make(map[schema.GroupKind]*v1beta1.CustomResourceDefinition)
			if tc.crd != nil {
				m[schema.GroupKind{}] = tc.crd
			}
			want := importer.StubbedCRDClusterConfigInfo(m)
			if diff := cmp.Diff(want, crdInfo, cmp.AllowUnexported(importer.CRDClusterConfigInfo{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
