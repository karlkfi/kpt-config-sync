package clusterconfig

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/testing/testoutput"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func servedStorage(t *testing.T, served, storage bool) core.MetaMutator {
	return func(o core.Object) {
		crd, ok := o.(*v1beta1.CustomResourceDefinition)
		if !ok {
			t.Fatalf("not a v1beta1.CRD: %T", o)
		}
		crd.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{
			{
				Served:  served,
				Storage: storage,
			},
		}
	}
}

func TestGetCRDs(t *testing.T) {
	testCases := []struct {
		name string
		objs []core.Object
		want []*v1beta1.CustomResourceDefinition
	}{
		{
			name: "No CRDs",
			want: []*v1beta1.CustomResourceDefinition{},
		},
		{
			name: "v1Beta1 CRD",
			objs: []core.Object{
				fake.CustomResourceDefinitionV1Beta1Unstructured(),
			},
			want: []*v1beta1.CustomResourceDefinition{
				fake.CustomResourceDefinitionV1Beta1Object(servedStorage(t, true, true)),
			},
		},
		{
			name: "v1 CRD",
			objs: []core.Object{
				fake.ToCustomResourceDefinitionV1Object(fake.CustomResourceDefinitionV1Beta1Object()),
			},
			want: []*v1beta1.CustomResourceDefinition{
				fake.CustomResourceDefinitionV1Beta1Object(servedStorage(t, true, true)),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decoder := decode.NewGenericResourceDecoder(scheme.Scheme)
			cc := testoutput.ClusterConfig(tc.objs...)
			actual, err := GetCRDs(decoder, cc)

			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.want, actual, cmpopts.EquateEmpty()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
