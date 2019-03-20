package repo

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CurrentVersion is the version of the format for the ConfigManagement Repo.
const CurrentVersion = "0.2.0"

// Default returns a default Repo in case one is not defined in the source of truth.
func Default() *v1.Repo {
	return &v1.Repo{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds.Repo().Kind,
			APIVersion: kinds.Repo().GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "repo",
		},
		Spec: v1.RepoSpec{
			Version: CurrentVersion,
		},
	}
}
