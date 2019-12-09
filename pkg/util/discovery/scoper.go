package discovery

import (
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// IsNamespaced is true if the type is Namespaced, false otherwise.
type IsNamespaced bool

const (
	// ClusterScope is an object scoped to the cluster
	ClusterScope = IsNamespaced(false)
	// NamespaceScope is an object scoped to namespace
	NamespaceScope = IsNamespaced(true)
)

// Scoper wraps a map from GroupKinds to IsNamespaced.
type Scoper map[schema.GroupKind]IsNamespaced

// GetObjectScope implements Scoper
func (s *Scoper) GetObjectScope(o id.Resource) (IsNamespaced, status.Error) {
	if s == nil {
		return false, status.InternalError("missing Scoper")
	}
	if scope, hasScope := (*s)[o.GroupVersionKind().GroupKind()]; hasScope {
		return scope, nil
	}
	return false, UnknownObjectKindError(o)
}

// GetGroupKindScope returns whether the type is namespace-scoped or cluster-scoped.
// Returns an error if the GroupKind is unknown to the cluster.
func (s *Scoper) GetGroupKindScope(gk schema.GroupKind) (IsNamespaced, status.Error) {
	if s == nil {
		return false, status.InternalError("missing Scoper")
	}
	if scope, hasScope := (*s)[gk]; hasScope {
		return scope, nil
	}
	return false, UnknownGroupKindError(gk)
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
		(*s)[gks.GroupKind] = gks.IsNamespaced
	}
}

// GroupKindScope is a Kubernetes type, and whether it is Namespaced.
type GroupKindScope struct {
	schema.GroupKind
	IsNamespaced
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
		GroupKind:    gk,
		IsNamespaced: scope,
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
			result = append(result, GroupKindScope{
				GroupKind:    gk,
				IsNamespaced: IsNamespaced(resource.Namespaced),
			})
		}
	}

	return result, nil
}

// UnknownKindErrorCode is the error code for UnknownObjectKindError
const UnknownKindErrorCode = "1021" // Impossible to create consistent example.

var unknownKindError = status.NewErrorBuilder(UnknownKindErrorCode)

// UnknownObjectKindError reports that an object declared in the repo does not have a definition in the cluster.
func UnknownObjectKindError(resource id.Resource) status.Error {
	return unknownKindError.
		Sprintf("No CustomResourceDefinition is defined for the type %q in the cluster. "+
			"\nResource types that are not native Kubernetes objects must have a CustomResourceDefinition.",
			resource.GroupVersionKind().GroupKind()).
		BuildWithResources(resource)
}

// UnknownGroupKindError reports that a GroupKind is not defined on the cluster, so we can't sync it.
func UnknownGroupKindError(gk schema.GroupKind) status.Error {
	return unknownKindError.
		Sprintf("No CustomResourceDefinition is defined for the type %q in the cluster. "+
			"\nResource types that are not native Kubernetes objects must have a CustomResourceDefinition.", gk).
		Build()
}
