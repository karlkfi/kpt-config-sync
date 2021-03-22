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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func preserveUnknownFields(t *testing.T, preserved bool) core.MetaMutator {
	return func(o client.Object) {
		switch crd := o.(type) {
		case *apiextensionsv1beta1.CustomResourceDefinition:
			crd.Spec.PreserveUnknownFields = &preserved
		case *apiextensionsv1.CustomResourceDefinition:
			crd.Spec.PreserveUnknownFields = preserved
		default:
			t.Fatalf("not a v1beta1.CRD or v1.CRD: %T", o)
		}
	}
}

func TestGetCRDs(t *testing.T) {
	testCases := []struct {
		name string
		objs []client.Object
		want []*apiextensionsv1beta1.CustomResourceDefinition
	}{
		{
			name: "No CRDs",
			want: []*apiextensionsv1beta1.CustomResourceDefinition{},
		},
		{
			name: "v1Beta1 CRD",
			objs: []client.Object{
				fake.CustomResourceDefinitionV1Beta1Unstructured(),
			},
			want: []*apiextensionsv1beta1.CustomResourceDefinition{
				fake.CustomResourceDefinitionV1Beta1Object(),
			},
		},
		{
			name: "v1 CRD",
			objs: []client.Object{
				fake.CustomResourceDefinitionV1Object(),
			},
			want: []*apiextensionsv1beta1.CustomResourceDefinition{
				fake.CustomResourceDefinitionV1Beta1Object(preserveUnknownFields(t, false)),
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

			if diff := cmp.Diff(tc.want, actual, cmpopts.EquateEmpty(),
				cmpopts.IgnoreFields(apiextensionsv1beta1.CustomResourceDefinition{}, "TypeMeta")); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func generateMalformedCRD(t *testing.T) client.Object {
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
		obj     client.Object
		wantErr status.Error
	}{
		{
			name:    "well-formed v1beta1 CRD",
			obj:     fake.CustomResourceDefinitionV1Beta1Object(),
			wantErr: nil,
		},
		{
			name:    "well-formed v1 CRD",
			obj:     fake.CustomResourceDefinitionV1Object(),
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
