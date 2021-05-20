package status

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResourceState(t *testing.T) {
	resources := []resourceState{
		{
			Group:     "apps",
			Kind:      "Deployment",
			Namespace: "bookstore",
			Name:      "test",
			Status:    "CURRENT",
		},
		{
			Group:     "",
			Kind:      "Service",
			Namespace: "bookstore",
			Name:      "test",
			Status:    "CURRENT",
		},
		{
			Kind:      "Service",
			Namespace: "gamestore",
			Name:      "test",
			Status:    "CURRENT",
		},
		{
			Group:  "rbac.authorization.k8s.io",
			Kind:   "ClusterRole",
			Name:   "test",
			Status: "CURRENT",
		},
	}
	sort.Sort(byNamespaceAndType(resources))

	expected := []resourceState{
		{
			Group:  "rbac.authorization.k8s.io",
			Kind:   "ClusterRole",
			Name:   "test",
			Status: "CURRENT",
		},
		{
			Group:     "apps",
			Kind:      "Deployment",
			Namespace: "bookstore",
			Name:      "test",
			Status:    "CURRENT",
		},
		{
			Group:     "",
			Kind:      "Service",
			Namespace: "bookstore",
			Name:      "test",
			Status:    "CURRENT",
		},
		{
			Kind:      "Service",
			Namespace: "gamestore",
			Name:      "test",
			Status:    "CURRENT",
		},
	}
	if diff := cmp.Diff(expected, resources); diff != "" {
		t.Error(diff)
	}
}
