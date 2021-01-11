package clusterconfig

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// GetCRDs returns the names and CustomResourceDefinitions of the CRDs in ClusterConfig.
func GetCRDs(decoder decode.Decoder, clusterConfig *v1.ClusterConfig) ([]*apiextensionsv1beta1.
	CustomResourceDefinition, status.Error) {
	if clusterConfig == nil {
		return nil, nil
	}

	gvkrs, err := decoder.DecodeResources(clusterConfig.Spec.Resources)
	if err != nil {
		return nil, status.APIServerErrorf(err, "could not deserialize CRD in %s", v1.CRDClusterConfigName)
	}

	crdMap := make(map[string]*apiextensionsv1beta1.CustomResourceDefinition)
	for gvk, unstructureds := range gvkrs {
		if gvk.GroupKind() != kinds.CustomResourceDefinition() {
			return nil, status.APIServerErrorf(err, "%s contains non-CRD resources: %v", v1.CRDClusterConfigName, gvk)
		}
		for _, u := range unstructureds {
			obj, err := reader.AsStruct(u)
			if err != nil {
				return nil, status.InternalErrorBuilder.Wrap(err).
					BuildWithResources(u)
			}

			cObj, ok := obj.(core.Object)
			if !ok {
				return nil, status.InternalErrorBuilder.
					Sprintf("%T is not a valid CRD type", obj).
					BuildWithResources(u)
			}

			crd, err := AsCRD(cObj)
			if err != nil {
				return nil, MalformedCRDError(err, u)
			}
			crdMap[crd.GetName()] = crd
		}
	}

	var crds []*apiextensionsv1beta1.CustomResourceDefinition
	for _, crd := range crdMap {
		crds = append(crds, crd)
	}
	return crds, nil
}

// MalformedCRDErrorCode is the error code for MalformedCRDError.
const MalformedCRDErrorCode = "1065"

var malformedCRDErrorBuilder = status.NewErrorBuilder(MalformedCRDErrorCode)

// MalformedCRDError reports a malformed CRD.
func MalformedCRDError(err error, obj id.Resource) status.Error {
	return malformedCRDErrorBuilder.Wrap(err).
		Sprint("malformed CustomResourceDefinition").
		BuildWithResources(obj)
}

// AsCRD returns the typed version of the CustomResourceDefinition passed in.
func AsCRD(o core.Object) (*apiextensionsv1beta1.CustomResourceDefinition, status.Error) {
	switch crd := o.(type) {
	case *apiextensionsv1beta1.CustomResourceDefinition:
		return crd, nil
	case *apiextensionsv1.CustomResourceDefinition:
		return AsV1Beta1CRD(crd)
	}

	return nil, MalformedCRDError(fmt.Errorf("could not generate a CRD from %T: %#v", o, o), o)
}

// AsV1Beta1CRD converts a v1 CRD to a v1beta1 CRD.
func AsV1Beta1CRD(crdV1 *apiextensionsv1.CustomResourceDefinition) (*apiextensionsv1beta1.CustomResourceDefinition, status.Error) {
	// Use the apiextensions conversion functions to convert to a v1beta1 CRD.
	crd := &apiextensions.CustomResourceDefinition{}
	err := apiextensionsv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(crdV1, crd, nil)
	if err != nil {
		return nil, MalformedCRDError(errors.Wrapf(err, "could not generate a v1 CRD from %T: %#v", crdV1, crdV1), crdV1)
	}
	crdV1Beta1 := &apiextensionsv1beta1.CustomResourceDefinition{}
	err = apiextensionsv1beta1.Convert_apiextensions_CustomResourceDefinition_To_v1beta1_CustomResourceDefinition(crd, crdV1Beta1, nil)
	if err != nil {
		return nil, MalformedCRDError(errors.Wrapf(err, "could not generate a v1beta1 CRD from %T: %#v", crd, crd), crdV1)
	}
	return crdV1Beta1, nil
}
