package initialize

import (
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/google/nomos/pkg/policyimporter/filesystem/nomospath"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	readmeFile         = "README.md"
	rootReadmeContents = `# GKE Policy Management Directory

This is the root directory for GKE Policy Management for your cluster.

* See [system/](system/README.md) for system configuration.
* See [cluster/](cluster/README.md) for cluster-scoped resources.
* See [clusterregistry/](clusterregistry/README.md) for clusterregistry-scoped resources.
* See [namespaces/](namespaces/README.md) for namespace-scoped resources.
`
	systemReadmeContents = `# System

This directory contains system configs such as the repo version and how resources are synced.
`
)

var defaultRepo = ast.FileObject{
	Path: nomospath.FromSlash("system/repo.yaml"),
	Object: &v1.Repo{
		TypeMeta: metav1.TypeMeta{
			Kind:       kinds.Repo().Kind,
			APIVersion: kinds.Repo().GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "repo",
		},
		Spec: v1.RepoSpec{
			Version: "0.1.0",
		},
	},
}
