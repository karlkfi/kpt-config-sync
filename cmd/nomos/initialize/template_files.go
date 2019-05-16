package initialize

import (
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/util/repo"
)

const (
	readmeFile         = "README.md"
	rootReadmeContents = `# Anthos Configuration Management Directory

This is the root directory for Anthos Configuration Management.

See [our documentation](https://cloud.google.com/anthos-config-management/docs/repo) for how to use each subdirectory.
`
	systemReadmeContents = `# System

This directory contains system configs such as the repo version and how resources are synced.
`
)

var defaultRepo = ast.FileObject{
	Path:   cmpath.FromSlash("system/repo.yaml"),
	Object: repo.Default(),
}
