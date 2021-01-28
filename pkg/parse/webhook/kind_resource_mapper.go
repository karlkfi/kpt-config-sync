package webhook

import (
	"github.com/google/nomos/pkg/status"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// kindResourceMapper maps GVKs to GVRs.
type kindResourceMapper map[schema.GroupVersionKind]schema.GroupVersionResource

// newKindResourceMapper creates a mapping from GVK to GVR from the API
// Resources known by the API Server.
//
// Returns an error if the APIResourceLists returned by the API Server are
// corrupted. This should be rare or never happen.
func newKindResourceMapper(lists []*metav1.APIResourceList) (kindResourceMapper, status.MultiError) {
	mapper := make(kindResourceMapper)

	var errs status.MultiError
	for _, list := range lists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			// This shouldn't happen.
			errs = status.Append(errs, status.APIServerErrorf(err,
				"API Server returned invalid GroupVersion %q", list.GroupVersion))
		}
		for _, resource := range list.APIResources {
			gvk := gv.WithKind(resource.Kind)
			gvr := gv.WithResource(resource.Name)
			mapper[gvk] = gvr
		}
	}

	if errs != nil {
		return nil, errs
	}
	return mapper, nil
}

func (m *kindResourceMapper) addV1CRDs(crds []apiextensionsv1.CustomResourceDefinition) {
	for _, crd := range crds {
		group := crd.Spec.Group
		kind := crd.Spec.Names.Kind
		gk := schema.GroupKind{Group: group, Kind: kind}
		name := crd.Spec.Names.Plural
		gr := schema.GroupResource{Group: group, Resource: name}
		for _, v := range crd.Spec.Versions {
			if v.Served {
				gvk := gk.WithVersion(v.Name)
				gvr := gr.WithVersion(v.Name)
				(*m)[gvk] = gvr
			}
		}
	}
}

func (m *kindResourceMapper) addV1Beta1CRDs(crds []apiextensionsv1beta1.CustomResourceDefinition) {
	for _, crd := range crds {
		group := crd.Spec.Group
		kind := crd.Spec.Names.Kind
		gk := schema.GroupKind{Group: group, Kind: kind}
		name := crd.Spec.Names.Plural
		gr := schema.GroupResource{Group: group, Resource: name}
		for _, v := range crd.Spec.Versions {
			if v.Served {
				gvk := gk.WithVersion(v.Name)
				gvr := gr.WithVersion(v.Name)
				(*m)[gvk] = gvr
			}
		}
		// If using the deprecated spec.version field.
		//noinspection GoDeprecation
		if v := crd.Spec.Version; v != "" {
			gvk := gk.WithVersion(v)
			gvr := gr.WithVersion(v)
			(*m)[gvk] = gvr
		}
	}
}

func toGVRs(mapper kindResourceMapper, gvks []schema.GroupVersionKind) ([]schema.GroupVersionResource, status.MultiError) {
	gvrs := make([]schema.GroupVersionResource, len(gvks))

	var errs status.MultiError
	for i, gvk := range gvks {
		var found bool
		gvrs[i], found = mapper[gvk]
		if !found {
			// This means we've made a mistake in parsing logic, as we should have
			// already validated that all declared types are on the API Server.
			errs = status.Append(errs,
				status.InternalErrorf("API Server does not have mapping for parsed kind %v", gvk))
		}
	}
	return gvrs, errs
}
