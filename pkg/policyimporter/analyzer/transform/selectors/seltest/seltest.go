// Package seltest has helper functions for creating selector objects.
package seltest

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ProdNamespaceSelector is a selector that selects a production namespace with an "env" annotation.
var ProdNamespaceSelector = v1alpha1.NamespaceSelector{
	Spec: v1alpha1.NamespaceSelectorSpec{
		Selector: metav1.LabelSelector{
			MatchLabels: map[string]string{"env": "prod"}}}}

// SensitiveNamespaceSelector is a selector that selects a production namespace with a sensitivity annotation.
var SensitiveNamespaceSelector = v1alpha1.NamespaceSelector{
	Spec: v1alpha1.NamespaceSelectorSpec{
		Selector: metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "privacy", Operator: metav1.LabelSelectorOpIn, Values: []string{"sensitive", "restricted"}}},
			MatchLabels:      map[string]string{"env": "prod"}}}}

// Cluster creates a cluster object for test.
func Cluster(name string, labels map[string]string) clusterregistry.Cluster {
	return clusterregistry.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}

// Selector creates a named cluster selector object for test.
func Selector(name string, selector metav1.LabelSelector) v1alpha1.ClusterSelector {
	return v1alpha1.ClusterSelector{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ClusterSelectorSpec{
			Selector: selector,
		},
	}
}

// Annotated creates a general annotated object for test.
func Annotated(a map[string]string) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Name:        "obj",
		Annotations: a,
	}
}
