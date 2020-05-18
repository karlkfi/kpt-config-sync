package repo

import (
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CurrentVersion is the version of the format for the ConfigManagement Repo.
const CurrentVersion = "1.0.0"

// Default returns a default Repo in case one is not defined in the source of truth.
func Default() *v1.Repo {
	return setTypeMeta(&v1.Repo{
		ObjectMeta: metav1.ObjectMeta{
			Name: "repo",
		},
		Spec: v1.RepoSpec{
			Version: CurrentVersion,
		},
	})
}

// setTypeMeta sets the fields for TypeMeta since they are usually unset when fetching a Repo from
// kubebuilder cache for some reason.
func setTypeMeta(r *v1.Repo) *v1.Repo {
	r.TypeMeta = metav1.TypeMeta{
		Kind:       kinds.Repo().Kind,
		APIVersion: kinds.Repo().GroupVersion().String(),
	}
	return r
}
