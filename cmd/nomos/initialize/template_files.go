package initialize

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/util/repo"
)

const (
	readmeFile         = "README.md"
	rootReadmeContents = `# CSP Configuration Management Directory

This is the root directory for CSP Configuration Management for your cluster.

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
	Path:   cmpath.FromSlash("system/repo.yaml"),
	Object: repo.Default(),
}
