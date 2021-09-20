// Package filesystem provides functionality to read Kubernetes objects from a filesystem tree
// and converting them to Nomos Custom Resource Definition objects.
package filesystem

import (
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement/v1/repo"
	"github.com/google/nomos/pkg/importer/analyzer/ast"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/status"
)

// Parser reads files on disk and builds Nomos Config objects to be reconciled by the Syncer.
type Parser struct {
	reader reader.Reader
}

var _ ConfigParser = &Parser{}

// NewParser creates a new Parser using the given Reader and parser options.
func NewParser(reader reader.Reader) *Parser {
	return &Parser{
		reader: reader,
	}
}

// Parse parses file tree rooted at root and builds policy CRDs from supported Kubernetes policy resources.
// Resources are read from the following directories:
//
// clusterName is the spec.clusterName of the cluster's ConfigManagement.
// enableAPIServerChecks, if true, contacts the API Server if it is unable to
//   determine whether types are namespace- or cluster-scoped.
// getSyncedCRDs is a callback that returns the CRDs on the API Server.
// filePaths is the list of absolute file paths to parse and the absolute and
//   relative paths of the Nomos root.
// It is an error for any files not to be present.
func (p *Parser) Parse(filePaths reader.FilePaths) ([]ast.FileObject, status.MultiError) {
	return p.reader.Read(filePaths)
}

// filterTopDir returns the set of files contained in the top directory of root
//   along with the absolute and relative paths of root.
// Assumes all files are within root.
func filterTopDir(filePaths reader.FilePaths, topDir string) reader.FilePaths {
	rootSplits := filePaths.RootDir.Split()
	var result []cmpath.Absolute
	for _, f := range filePaths.Files {
		if f.Split()[len(rootSplits)] != topDir {
			continue
		}
		result = append(result, f)
	}
	return reader.FilePaths{
		RootDir:   filePaths.RootDir,
		PolicyDir: filePaths.PolicyDir,
		Files:     result,
	}
}

// ReadClusterRegistryResources reads the manifests declared in clusterregistry/ for hierarchical format.
// For unstructured format, it reads all files.
func (p *Parser) ReadClusterRegistryResources(filePaths reader.FilePaths, format SourceFormat) ([]ast.FileObject, status.MultiError) {
	if format == SourceFormatHierarchy {
		return p.reader.Read(filterTopDir(filePaths, repo.ClusterRegistryDir))
	}
	return p.reader.Read(filePaths)
}

// ReadClusterNamesFromSelector returns the list of cluster names specified in the `cluster-name-selector` annotation.
func (p *Parser) ReadClusterNamesFromSelector(filePaths reader.FilePaths) ([]string, status.MultiError) {
	var clusters []string
	objs, err := p.Parse(filePaths)
	if err != nil {
		return clusters, err
	}

	for _, obj := range objs {
		if annotation, found := obj.GetAnnotations()[metadata.ClusterNameSelectorAnnotationKey]; found {
			names := strings.Split(annotation, ",")
			for _, name := range names {
				clusters = append(clusters, strings.TrimSpace(name))
			}
		}
	}
	return clusters, nil
}
