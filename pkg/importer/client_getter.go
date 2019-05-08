package importer

import (
	"fmt"

	"github.com/google/nomos/pkg/client/action"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var _ apimeta.RESTMapper = CRDAwareRestMapper{}

// CRDAwareRestMapper generates rest mappings based on the discovery API and a set of CRDs.
type CRDAwareRestMapper struct {
	apimeta.RESTMapper
	mappings    map[schema.GroupVersionKind]*apimeta.RESTMapping
	stubMissing bool
}

// NewCRDAwareRestMapper returns a new CRDAwareRestMapper.
func NewCRDAwareRestMapper(delegate apimeta.RESTMapper, stubMissing bool, crds ...*v1beta1.CustomResourceDefinition) apimeta.
	RESTMapper {
	return CRDAwareRestMapper{
		RESTMapper:  delegate,
		stubMissing: stubMissing,
		mappings:    crdRestMappings(crds),
	}
}

// crdRestMappings converts CRDs to the corresponding RESTMappings for the Custom Resources they describe.
func crdRestMappings(crds []*v1beta1.CustomResourceDefinition) map[schema.GroupVersionKind]*apimeta.RESTMapping {
	mappings := make(map[schema.GroupVersionKind]*apimeta.RESTMapping)
	for _, crd := range crds {
		setVersion := func(version string) {
			gvk := schema.GroupVersionKind{
				Group:   crd.Spec.Group,
				Version: version,
				Kind:    crd.Spec.Names.Kind,
			}
			scope := apimeta.RESTScopeRoot
			if crd.Spec.Scope == v1beta1.NamespaceScoped {
				scope = apimeta.RESTScopeNamespace
			}
			mappings[gvk] = &apimeta.RESTMapping{
				Resource:         gvk.GroupVersion().WithResource(crd.Spec.Names.Plural),
				GroupVersionKind: gvk,
				Scope:            scope,
			}
		}

		for _, v := range crd.Spec.Versions {
			if !v.Served {
				continue
			}
			setVersion(v.Name)
		}

		if v := crd.Spec.Version; v != "" {
			setVersion(v)
		}
	}
	return mappings
}

// RESTMapping implements RESTMapper.
func (e CRDAwareRestMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*apimeta.RESTMapping, error) {
	m, err := e.RESTMapper.RESTMapping(gk, versions...)
	if err == nil {
		return m, nil
	}

	for _, v := range versions {
		m, ok := e.mappings[gk.WithVersion(v)]
		if !ok {
			continue
		}
		return m, nil
	}

	if !e.stubMissing {
		return nil, fmt.Errorf("no mapping found for %#v, versions=%v", gk, versions)
	}

	// Return an API mapping based on what is requested, on a best-effort basis.
	var version string
	if len(versions) > 0 {
		version = versions[0]
	}
	gvk := gk.WithVersion(version)
	return &apimeta.RESTMapping{
		Resource:         gvk.GroupVersion().WithResource(action.LowerPlural(gvk.Kind)),
		GroupVersionKind: gvk,
		// we're should only be looking at a cluster-scoped resource,
		// since this fallback only comes up for custom resources that do not have a CRD already added to the cluster.
		Scope: apimeta.RESTScopeRoot,
	}, nil
}

var _ genericclioptions.RESTClientGetter = CRDAwareClientGetter{}

// CRDAwareClientGetter returns a rest mapper that generates mappings based upon a set of CRDs.
// All other functionality is based on the delegate RESTClientGetter this struct contains.
type CRDAwareClientGetter struct {
	genericclioptions.RESTClientGetter
	crds        []*v1beta1.CustomResourceDefinition
	stubMissing bool
}

// NewFilesystemCRDAwareClientGetter returns a new CRDAwareClientGetter.
func NewFilesystemCRDAwareClientGetter(g genericclioptions.RESTClientGetter, stubMissing bool,
	crds ...*v1beta1.CustomResourceDefinition) CRDAwareClientGetter {
	return CRDAwareClientGetter{
		RESTClientGetter: g,
		stubMissing:      stubMissing,
		crds:             crds,
	}
}

// ToRESTMapper implements RESTClientGetter.
func (cg CRDAwareClientGetter) ToRESTMapper() (apimeta.RESTMapper, error) {
	rm, err := cg.RESTClientGetter.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	return NewCRDAwareRestMapper(rm, cg.stubMissing, cg.crds...), nil
}
