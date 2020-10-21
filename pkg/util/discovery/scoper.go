package discovery

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/status"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ScopeType is the namespace/cluster scope of a particular GroupKind.
type ScopeType string

const (
	// ClusterScope is an object scoped to the cluster.
	ClusterScope = ScopeType("Cluster")
	// NamespaceScope is an object scoped to namespace.
	NamespaceScope = ScopeType("Namespace")
	// UnknownScope means we don't know the scope of the type.
	UnknownScope = ScopeType("Unknown")
)

// NewScoper returns a Scoper for determining whether objects/types are Namespaced.
//
// errOnUnknown is whether to return an error if the scope for an object or type
// is either explicitly marked Unknown or is not in the initially pased map.
func NewScoper(scopes map[schema.GroupKind]ScopeType, errOnUnknown bool) Scoper {
	return Scoper{
		scope:        scopes,
		errOnUnknown: errOnUnknown,
	}
}

// Scoper wraps a map from GroupKinds to ScopeType.
type Scoper struct {
	scope        map[schema.GroupKind]ScopeType
	errOnUnknown bool
}

// GetObjectScope implements Scoper
func (s *Scoper) GetObjectScope(o id.Resource) (ScopeType, status.Error) {
	scope, err := s.GetGroupKindScope(o.GroupVersionKind().GroupKind())
	if err != nil {
		// Make the error specific to the object.
		return scope, UnknownObjectKindError(o)
	}
	return scope, nil
}

// GetGroupKindScope returns whether the type is namespace-scoped or cluster-scoped.
// Returns an error if the GroupKind is unknown to the cluster.
func (s *Scoper) GetGroupKindScope(gk schema.GroupKind) (ScopeType, status.Error) {
	if s == nil {
		return UnknownScope, status.InternalError("missing Scoper")
	}
	if scope, hasScope := s.scope[gk]; hasScope && scope != UnknownScope {
		return scope, nil
	}
	// We weren't able to get the scope for this type.
	if s.errOnUnknown {
		return UnknownScope, UnknownGroupKindError(gk)
	}
	return UnknownScope, nil
}

// HasScopesFor returns true if the Scoper knows the scopes for every type in the
// passed slice of FileObjects.
//
// While this could be made more general, it requests a slice of FileObjects
// as this is a convenience method.
func (s *Scoper) HasScopesFor(objects []ast.FileObject) bool {
	for _, o := range objects {
		if _, exists := s.scope[o.GroupVersionKind().GroupKind()]; !exists {
			return false
		}
	}
	return true
}

// AddCustomResources updates Scoper with custom resource metadata from the provided CustomResourceDefinitions.
// It does not replace anything that already exists in the Scoper.
func (s *Scoper) AddCustomResources(crds []*v1beta1.CustomResourceDefinition) {
	gkss := ScopesFromCRDs(crds)
	s.add(gkss)
}

func (s *Scoper) add(gkss []GroupKindScope) {
	for _, gks := range gkss {
		// Explicitly do not overwrite scopes that have already been added.
		if _, hasGK := s.scope[gks.GroupKind]; hasGK {
			continue
		}
		s.scope[gks.GroupKind] = gks.ScopeType
	}
}

// GroupKindScope is a Kubernetes type, and whether it is Namespaced.
type GroupKindScope struct {
	schema.GroupKind
	ScopeType
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
		ScopeType: scope,
	}
}

// AddAPIResourceLists adds APIResourceLists retrieved from an API Server to
// the Scoper, taking care to not overwrite explicitly-defined or
// implicitly-known scopes.
func (s *Scoper) AddAPIResourceLists(resourceLists []*metav1.APIResourceList) status.MultiError {
	// Collect all of the server-declared scopes.
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

	// The resource lists contain an invalid GroupVersion. There isn't a clean
	// way to recover from this.
	//
	// This could be more lenient, e.g. if the type we had an error on isn't
	// actually required, we could ignore it.
	if errs != nil {
		return errs
	}

	// Define scopes for all types on the APIServer for which there are not
	// already known scopes.
	s.add(allGKSs)
	return nil
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
			gks := GroupKindScope{
				GroupKind: schema.GroupKind{
					Group: groupVersion.Group,
					Kind:  resource.Kind,
				}}
			if resource.Namespaced {
				gks.ScopeType = NamespaceScope
			} else {
				gks.ScopeType = ClusterScope
			}
			result = append(result, gks)
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
