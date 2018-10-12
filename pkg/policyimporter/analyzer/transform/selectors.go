package transform

// Shared code for NamespaceSelector and ClusterSelector.

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// asPopulatedSelector returns a known valid and nonempty label selector.
func asPopulatedSelector(labelselector *metav1.LabelSelector) (labels.Selector, error) {
	selector, err := metav1.LabelSelectorAsSelector(labelselector)
	if err != nil {
		return nil, fmt.Errorf("invalid label selector: %v", err)
	}
	if selector.Empty() {
		return nil, fmt.Errorf("empty label selector")
	}
	return selector, nil
}

// isSelected returns true if the labels match the selector.
func isSelected(l map[string]string, selector labels.Selector) bool {
	return selector.Matches(labels.Set(l))
}
