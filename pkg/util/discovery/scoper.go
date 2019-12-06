package discovery

import (
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectScope is the return type for APIInfo.GetScope
type ObjectScope string

const (
	// ClusterScope is an object scoped to the cluster
	ClusterScope = ObjectScope("cluster")
	// NamespaceScope is an object scoped to namespace
	NamespaceScope = ObjectScope("namespace")
	// UnknownScope is returned if the object does not exist in APIInfo
	UnknownScope = ObjectScope("unknown")
)

// Scoper wraps a map from GroupKinds to ObjectScope.
type Scoper map[schema.GroupKind]ObjectScope

// GetScope implements Scoper
func (s *Scoper) GetScope(gk schema.GroupKind) ObjectScope {
	if s == nil {
		return UnknownScope
	}
	if scope, hasScope := (*s)[gk]; hasScope {
		return scope
	}
	return UnknownScope
}

// AddCustomResources updates Scoper with custom resource metadata from the provided CustomResourceDefinitions.
// It does not replace anything that already exists in the Scoper.
func (s *Scoper) AddCustomResources(crds []*v1beta1.CustomResourceDefinition) {
	gkss := ScopesFromCRDs(crds)
	s.add(gkss)
}

func (s *Scoper) add(gkss []GroupKindScope) {
	for _, gks := range gkss {
		// Explicitly do not overwrite scopes as specified on the APIServer.
		if _, hasGK := (*s)[gks.GroupKind]; hasGK {
			continue
		}
		(*s)[gks.GroupKind] = gks.Scope
	}
}

// GroupKindScope is a Kubernetes type, and whether it is Namespaced.
type GroupKindScope struct {
	schema.GroupKind
	Scope ObjectScope
}

// ScopesFromCRDs extracts the scopes declared in all passed CRDs.
func ScopesFromCRDs(crds []*v1beta1.CustomResourceDefinition) []GroupKindScope {
	var result []GroupKindScope
	for _, crd := range crds {
		if !isServed(crd) {
			continue
		}

		result = append(result, scopeFromCRD(crd))
	}
	return result
}

// isServed returns true if the CRD declares a version servable by the APIServer.
func isServed(crd *v1beta1.CustomResourceDefinition) bool {
	if crd.Spec.Version != "" {
		return true
	}
	for _, version := range crd.Spec.Versions {
		if version.Served {
			return true
		}
	}
	return false
}

func scopeFromCRD(crd *v1beta1.CustomResourceDefinition) GroupKindScope {
	// CRD Scope defaults to Namespaced
	scope := NamespaceScope
	if crd.Spec.Scope == v1beta1.ClusterScoped {
		scope = ClusterScope
	}

	gk := schema.GroupKind{
		Group: crd.Spec.Group,
		Kind:  crd.Spec.Names.Kind,
	}

	return GroupKindScope{
		GroupKind: gk,
		Scope:     scope,
	}
}

// NewScoperFromServerResources constructs a Scoper from a set of APIResourcesLists,
// and additional types for which the scope is known, e.g. CustomResourceDefinitions.
//
// The scopes in `additional` overwrite the server-side declared scopes.
func NewScoperFromServerResources(resourceLists []*metav1.APIResourceList, additional ...GroupKindScope) (Scoper, status.MultiError) {
	var errs status.MultiError
	var allGKSs []GroupKindScope
	for _, list := range resourceLists {
		gkss, err := toGKSs(list)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		allGKSs = append(allGKSs, gkss...)
	}
	if errs != nil {
		return nil, errs
	}
	allGKSs = append(allGKSs, additional...)

	scoper := Scoper{}
	scoper.add(allGKSs)
	return scoper, nil
}

// toGVKSs flattens an APIResourceList to the set of GVKs and their respective ObjectScopes.
func toGKSs(lists ...*metav1.APIResourceList) ([]GroupKindScope, error) {
	var result []GroupKindScope

	for _, list := range lists {
		groupVersion, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "discovery client returned invalid GroupVersion: %q", list.GroupVersion)
		}

		for _, resource := range list.APIResources {
			gk := schema.GroupKind{
				Group: groupVersion.Group,
				Kind:  resource.Kind,
			}
			scope := ClusterScope
			if resource.Namespaced {
				scope = NamespaceScope
			}
			result = append(result, GroupKindScope{
				GroupKind: gk,
				Scope:     scope,
			})
		}
	}

	return result, nil
}
