package clusterconfig

import (
	"fmt"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/decode"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// GetCRDs returns the names and CustomResourceDefinitions of the CRDs in ClusterConfig.
func GetCRDs(decoder decode.Decoder, clusterConfig *v1.ClusterConfig) ([]*v1beta1.
	CustomResourceDefinition, status.Error) {
	if clusterConfig == nil {
		return nil, nil
	}

	gvkrs, err := decoder.DecodeResources(clusterConfig.Spec.Resources)
	if err != nil {
		return nil, status.APIServerErrorf(err, "could not deserialize CRD in %s", v1.CRDClusterConfigName)
	}

	crdMap := make(map[string]*v1beta1.CustomResourceDefinition)
	for gvk, rs := range gvkrs {
		if gvk != kinds.CustomResourceDefinitionV1Beta1() {
			return nil, status.APIServerErrorf(err, "%s contains non-CRD resources: %v", v1.CRDClusterConfigName, gvk)
		}
		for _, r := range rs {
			crd, err := unstructuredToCRD(r)
			if err != nil {
				return nil, status.APIServerErrorf(err, "could not deserialize CRD in %s", v1.CRDClusterConfigName)
			}
			crdMap[crd.Name] = crd
		}
	}

	var crds []*v1beta1.CustomResourceDefinition
	for _, crd := range crdMap {
		crds = append(crds, crd)
	}
	return crds, nil
}

// AsCRD returns the typed version of the CustomResourceDefinition passed in.
func AsCRD(o core.Object) (*v1beta1.CustomResourceDefinition, error) {
	if crd, ok := o.(*v1beta1.CustomResourceDefinition); ok {
		return crd, nil
	}
	// TODO(131853779): Should be able to always parse the typed CRD.
	if crd, ok := o.(*unstructured.Unstructured); ok {
		return unstructuredToCRD(crd)
	}
	return nil, fmt.Errorf("could not generate a CRD from %T: %#v", o, o)
}

// unstructuredToCRD returns the typed version of the CustomResourceDefinition of the Unstructured object.
func unstructuredToCRD(u *unstructured.Unstructured) (*v1beta1.CustomResourceDefinition, error) {
	name, _, nameErr := unstructured.NestedString(u.Object, "metadata", "name")
	if nameErr != nil {
		return nil, nameErr
	}
	group, _, groupErr := unstructured.NestedString(u.Object, "spec", "group")
	if groupErr != nil {
		return nil, groupErr
	}
	scope, _, scopeErr := unstructured.NestedString(u.Object, "spec", "scope")
	if scopeErr != nil {
		return nil, scopeErr
	}
	kind, _, kindErr := unstructured.NestedString(u.Object, "spec", "names", "kind")
	if kindErr != nil {
		return nil, kindErr
	}
	plural, _, pluralErr := unstructured.NestedString(u.Object, "spec", "names", "plural")
	if pluralErr != nil {
		return nil, pluralErr
	}
	singular, _, singularErr := unstructured.NestedString(u.Object, "spec", "names", "singular")
	if singularErr != nil {
		return nil, singularErr
	}
	shortNames, _, shortNamesErr := unstructured.NestedStringSlice(u.Object, "spec", "names", "shortNames")
	if shortNamesErr != nil {
		return nil, shortNamesErr
	}
	categories, _, categoriesErr := unstructured.NestedStringSlice(u.Object, "spec", "names", "categories")
	if categoriesErr != nil {
		return nil, categoriesErr
	}
	version, _, versionErr := unstructured.NestedString(u.Object, "spec", "version")
	if versionErr != nil {
		return nil, versionErr
	}
	versions, _, versionsErr := unstructured.NestedSlice(u.Object, "spec", "versions")
	if versionsErr != nil {
		return nil, versionsErr
	}

	crd := &v1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds.CustomResourceDefinitionV1Beta1().Kind,
			APIVersion: kinds.CustomResourceDefinitionV1Beta1().GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group: group,
			Scope: v1beta1.ResourceScope(scope),
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:       kind,
				Plural:     plural,
				Singular:   singular,
				ShortNames: shortNames,
				Categories: categories,
			},
		},
	}

	if len(versions) == 0 {
		crd.Spec.Versions = append(crd.Spec.Versions, v1beta1.CustomResourceDefinitionVersion{
			Name:    version,
			Served:  true,
			Storage: true,
		})
	}

	for _, v := range versions {
		crdVersion, _ := v.(map[string]interface{})
		name, _, nameErr := unstructured.NestedString(crdVersion, "name")
		if nameErr != nil {
			return nil, nameErr
		}
		served, _, servedErr := unstructured.NestedBool(crdVersion, "served")
		if servedErr != nil {
			return nil, servedErr
		}
		storage, _, storageErr := unstructured.NestedBool(crdVersion, "storage")
		if storageErr != nil {
			return nil, storageErr
		}
		crd.Spec.Versions = append(crd.Spec.Versions, v1beta1.CustomResourceDefinitionVersion{
			Name:    name,
			Served:  served,
			Storage: storage,
		})
	}

	return crd, nil
}
