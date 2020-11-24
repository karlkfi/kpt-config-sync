package clusterconfig

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/nomos/clientgen/apis/scheme"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/testing/testoutput"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func generateMalformedCRD(t *testing.T) core.Object {
	u := fake.CustomResourceDefinitionV1Beta1Unstructured()

	// the `spec.group` field should be a string
	// set it to a bool to construct a malformed CRD
	if err := unstructured.SetNestedField(u.Object, false, "spec", "group"); err != nil {
		t.Fatalf("failed to set the generation field: %T", u)
	}
	return u
}

func TestAsCRD(t *testing.T) {
	testCases := []struct {
		name    string
		obj     core.Object
		wantErr status.Error
	}{
		{
			name:    "well-formed CRD",
			obj:     fake.CustomResourceDefinitionV1Beta1Object(),
			wantErr: nil,
		},
		{
			name:    "mal-formed CRD",
			obj:     generateMalformedCRD(t),
			wantErr: malformedCRDErrorBuilder.Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := AsCRD(tc.obj)
			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("got error %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("got nil, want %v", tc.wantErr)
				} else {
					if !errors.Is(err, tc.wantErr) {
						t.Errorf("got error %v, want %v", err, tc.wantErr)
					}
				}
			}
		})
	}
}
