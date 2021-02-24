package hydrate

import (
	"strings"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/validate/objects"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ClusterSelectors hydrates the given Raw objects by performing cluster
// selection to filter out objects which are not specified for the current
// cluster.
func ClusterSelectors(objs *objects.Raw) status.MultiError {
	set, errs := buildHydratorSet(objs)
	if errs != nil {
		return errs
	}
	activeSelectors, errs := set.activeSelectors()
	if errs != nil {
		return errs
	}

	var filtered []ast.FileObject
	// We process namespaces first so that we can use their stateActive/stateInactive state
	// to do additional filtering on other resources  below.
	activeNamespaces := make(map[string]bool)
	for _, ns := range set.namespaces {
		objState, err := objectSelectionState(objs.ClusterName, ns, activeSelectors)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		if objState == stateActive {
			filtered = append(filtered, ns)
			activeNamespaces[ns.GetName()] = true
		} else {
			activeNamespaces[ns.GetName()] = false
		}
	}

	// Now process the rest of the resources.
	for _, res := range set.resources {
		// First filter out namespace-scoped resources that are in an stateInactive
		// namespace.
		if active, ok := activeNamespaces[res.GetNamespace()]; ok && !active {
			continue
		}
		// Now perform the same cluster selection filtering as before.
		objState, err := objectSelectionState(objs.ClusterName, res, activeSelectors)
		if err != nil {
			errs = status.Append(errs, err)
			continue
		}
		if objState == stateActive {
			filtered = append(filtered, res)
		}
	}

	if errs != nil {
		return errs
	}

	// We are done with Clusters and ClusterSelectors so we can filter them out
	// now as well.
	objs.Objects = filtered
	return nil
}

// buildHydratorSet splits the given Raw objects into important types (Cluster,
// ClusterSelector, Namespace) and populates a hydratorSet with them.
func buildHydratorSet(objs *objects.Raw) (*hydratorSet, status.MultiError) {
	set := &hydratorSet{}
	var errs status.MultiError
	for _, object := range objs.Objects {
		switch object.GroupVersionKind() {
		case kinds.Cluster():
			if object.GetName() == objs.ClusterName {
				if err := set.clusterObject(object); err != nil {
					errs = status.Append(errs, err)
				}
			}
		case kinds.ClusterSelector():
			if err := set.clusterSelectorObject(object); err != nil {
				errs = status.Append(errs, err)
			}
		case kinds.Namespace():
			set.namespaces = append(set.namespaces, object)
		default:
			set.resources = append(set.resources, object)
		}
	}
	return set, errs
}

func objectSelectionState(clusterName string, object ast.FileObject, activeSelectors map[string]bool) (state, status.Error) {
	legacyAnnotation, hasLegacyAnnotation := object.GetAnnotations()[v1.LegacyClusterSelectorAnnotationKey]
	inlineAnnotation, hasInlineAnnotation := object.GetAnnotations()[v1alpha1.ClusterNameSelectorAnnotationKey]

	switch {
	case hasLegacyAnnotation && hasInlineAnnotation:
		return stateUnknown, selectors.ClusterSelectorAnnotationConflictError(object)
	case !hasLegacyAnnotation && !hasInlineAnnotation:
		return stateActive, nil
	case hasInlineAnnotation:
		return stateFromInlineClusterSelector(clusterName, inlineAnnotation), nil
	default:
		active, known := activeSelectors[legacyAnnotation]
		if !known {
			return stateUnknown, selectors.ObjectHasUnknownClusterSelector(object, legacyAnnotation)
		}
		if active {
			return stateActive, nil
		}
		return stateInactive, nil
	}
}

// stateFromInlineClusterSelector returns the stateActive/stateInactive state for
// the object based on the inline cluster-name-selector annotation.
func stateFromInlineClusterSelector(clusterName, selector string) state {
	if len(clusterName) == 0 {
		return stateInactive
	}
	clusters := strings.Split(selector, ",")
	for _, cluster := range clusters {
		if strings.EqualFold(clusterName, strings.TrimSpace(cluster)) {
			return stateActive
		}
	}
	return stateInactive
}

type hydratorSet struct {
	cluster    *clusterregistry.Cluster
	selectors  []*v1.ClusterSelector
	namespaces []ast.FileObject
	resources  []ast.FileObject
}

func (h *hydratorSet) clusterObject(object ast.FileObject) status.Error {
	s, err := object.Structured()
	if err != nil {
		return err
	}
	h.cluster = s.(*clusterregistry.Cluster)
	return nil
}

func (h *hydratorSet) clusterSelectorObject(object ast.FileObject) status.Error {
	s, sErr := object.Structured()
	if sErr != nil {
		return sErr
	}

	h.selectors = append(h.selectors, s.(*v1.ClusterSelector))
	return nil
}

func (h *hydratorSet) activeSelectors() (map[string]bool, status.MultiError) {
	activeSels := make(map[string]bool)
	clusterLabels := labels.Set{}
	if h.cluster != nil {
		clusterLabels = h.cluster.Labels
	}

	var errs status.MultiError
	for _, s := range h.selectors {
		selector, err := metav1.LabelSelectorAsSelector(&s.Spec.Selector)
		if err != nil {
			errs = status.Append(errs, selectors.InvalidSelectorError(s, err))
			continue
		}
		if selector.Empty() {
			errs = status.Append(errs, selectors.EmptySelectorError(s))
			continue
		}
		activeSels[s.Name] = selector.Matches(clusterLabels)
	}

	return activeSels, errs
}

// state represents what we know about whether an object should be synced to the cluster
// based on the declared ClusterSelectors.
type state string

const (
	// stateActive represents objects that should be synced to the cluster.
	stateActive = state("stateActive")
	// stateInactive represents objects that should NOT be synced to the cluster.
	stateInactive = state("stateInactive")
	// stateUnknown is the value we use when we encounter a problem and can't
	// determine whether the object should be synced.
	stateUnknown = state("stateUnknown")
)
