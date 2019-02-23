package apiresource

import (
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Roles returns an APIResource describing a standard Role definition.
func Roles() metav1.APIResource {
	return metav1.APIResource{
		Name:       "roles",
		Namespaced: true,
		Group:      kinds.Role().Group,
		Version:    kinds.Role().Version,
		Kind:       kinds.Role().Kind,
		Verbs:      []string{"list"},
	}
}

// RoleBindings returns an APIResource describing a standard RoleBinding definition.
func RoleBindings() metav1.APIResource {
	return metav1.APIResource{
		Name:       "rolebindings",
		Namespaced: true,
		Group:      kinds.RoleBinding().Group,
		Version:    kinds.RoleBinding().Version,
		Kind:       kinds.RoleBinding().Kind,
		Verbs:      []string{"list"},
	}
}

// ClusterRoles returns an APIResource describing a standard ClusterRole definition.
func ClusterRoles() metav1.APIResource {
	return metav1.APIResource{
		Name:    "clusterroles",
		Group:   kinds.ClusterRole().Group,
		Version: kinds.ClusterRole().Version,
		Kind:    kinds.ClusterRole().Kind,
		Verbs:   []string{"list"},
	}
}

// Clusters returns an APIResource describing a standard Cluster definition.
func Clusters() metav1.APIResource {
	return metav1.APIResource{
		Name:    "clusters",
		Group:   kinds.Cluster().Group,
		Version: kinds.Cluster().Version,
		Kind:    kinds.Cluster().Kind,
		Verbs:   []string{"list"},
	}
}

// TokenReviews returns an APIResource describing a standard TokenReview definition.
func TokenReviews() metav1.APIResource {
	return metav1.APIResource{
		Name:    "tokenreviews",
		Group:   "authentication.k8s.io",
		Version: "",
		Kind:    "TokenReviews",
	}
}
