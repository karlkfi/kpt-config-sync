package examples

import (
	"errors"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/importer/analyzer/ast/asttesting"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/filesystem"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/testing/fake"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type exampleErrors map[string][]status.Error

// Generate generates example errors for documentation.
func Generate() map[string][]status.Error {
	// exampleErrors is a map of exampleErrors of each error type. For documentation purposes, i.e. for use
	// in the internal-only nomoserrors command.
	result := make(exampleErrors)

	// 1000
	result.add(status.InternalError("we made a mistake"))

	// 1001 is Deprecated.

	// 1002 is Deprecated.

	// 1003
	result.add(validation.IllegalNamespaceSubdirectoryError(node("namespaces/foo/bar"), node("namespaces/foo")))

	// 1004
	result.add(metadata.IllegalNamespaceAnnotationError(fake.Namespace("namespaces/foo")))

	// 1005
	result.add(nonhierarchical.IllegalManagementAnnotationError(fake.Role(), "invalid"))

	// 1006
	result.add(syntax.ObjectParseError(fake.Role()))

	// 1007
	result.add(validation.IllegalAbstractNamespaceObjectKindError(fake.Role()))

	// 1008 is Deprecated.

	// 1009
	result.add(metadata.IllegalMetadataNamespaceDeclarationError(
		fake.RoleAtPath("namespaces/foo/r.yaml", core.Namespace("bar")), "foo"))

	// 1010
	result.add(metadata.IllegalAnnotationDefinitionError(fake.Role(), []string{v1.ConfigManagementPrefix + "-annotation"}))

	// 1011
	result.add(metadata.IllegalLabelDefinitionError(fake.Role(), []string{v1.ConfigManagementPrefix + "-label"}))

	// 1012
	result.add(validation.NamespaceSelectorMayNotHaveAnnotation(fake.NamespaceSelector()))

	// 1013
	result.add(selectors.ObjectHasUnknownClusterSelector(fake.Role(), "undeclared-selector"))

	// 1014
	result.add(selectors.InvalidSelectorError("foo-selector", errors.New("some parse error")))

	// 1015 is Deprecated.

	// 1016
	result.add(filesystem.ConfigManagementNotInstalledError(errors.New("missing ConfigManagement")))

	// 1017
	result.add(system.MissingRepoError())

	// 1018 is Deprecated.

	// 1019
	result.add(metadata.IllegalTopLevelNamespaceError(fake.Namespace("namespaces")))

	// 1020
	result.add(metadata.InvalidNamespaceNameError(fake.Namespace("namespaces/foo", core.Name("")), "foo"))

	// 1021
	result.add(validation.UnknownObjectError(asttesting.NewFakeFileObject(schema.GroupVersionKind{
		Group:   "com.me",
		Version: "v1",
		Kind:    "Engineer",
	}, "namespaces/foo/engineer.yaml")))

	// 1022 is Deprecated.

	// 1023 is Deprecated.

	// 1024
	result.add(system.IllegalKindInSystemError(fake.RoleAtPath("system/role.yaml")))

	// 1025 is Deprecated.

	// 1026 is Deprecated.

	// 1027
	result.add(system.UnsupportedRepoSpecVersion(fake.Repo(fake.RepoVersion("")), "0.0.0"))

	// 1028
	result.add(syntax.ReservedDirectoryNameError(cmpath.FromSlash("namespaces/" + configmanagement.ControllerNamespace)))
	result.add(syntax.InvalidDirectoryNameError(cmpath.FromSlash("namespaces/ABC")))

	// 1029
	result.add(nonhierarchical.NamespaceCollisionError("qux",
		fake.Namespace("namespaces/foo/qux"),
		fake.Namespace("namespaces/bar/qux")))
	result.add(nonhierarchical.NamespaceMetadataNameCollisionError(kinds.Role().GroupKind(),
		"backend", "admin",
		fake.RoleAtPath("namespaces/backend/admin-1.yaml", core.Namespace("backend"), core.Name("admin")),
		fake.RoleAtPath("namespaces/backend/admin-2.yaml", core.Namespace("backend"), core.Name("admin")),
		fake.RoleAtPath("namespaces/backend/admin-3.yaml", core.Namespace("backend"), core.Name("admin")),
	))
	result.add(nonhierarchical.ClusterMetadataNameCollisionError(kinds.ClusterRole().GroupKind(),
		"cluster-admin",
		fake.ClusterRoleAtPath("cluster/admin-1.yaml", core.Name("cluster-admin")),
		fake.ClusterRoleAtPath("cluster/admin-2.yaml", core.Name("cluster-admin")),
	))

	// 1030
	result.add(semantic.MultipleSingletonsError(fake.Namespace("namespaces/foo"), fake.Namespace("namespaces/foo")))

	// 1031
	result.add(nonhierarchical.MissingObjectNameError(fake.Role(core.Name(""))))

	// 1032
	result.add(nonhierarchical.IllegalHierarchicalKind(fake.Repo()))

	// 1033
	result.add(syntax.IllegalSystemResourcePlacementError(fake.RepoAtPath("namespaces/repo.yaml")))

	// 1034
	result.add(nonhierarchical.IllegalNamespace(fake.Namespace("namespaces/" + configmanagement.ControllerNamespace)))
	result.add(nonhierarchical.IllegalNamespace(fake.RoleAtPath("namespaces/"+configmanagement.ControllerNamespace+"/role.yaml",
		core.Namespace("namespaces/"+configmanagement.ControllerNamespace))))

	// 1035 is Deprecated.

	// 1036
	result.add(nonhierarchical.InvalidMetadataNameError(fake.Role(core.Name("ABC"))))

	// 1037
	result.add(syntax.IllegalKindInClusterregistryError(fake.RoleAtPath("clusterregistry/role.yaml")))

	// 1038
	result.add(syntax.IllegalKindInNamespacesError(fake.ClusterRoleAtPath("namespaces/foo/cr.yaml")))

	// 1039
	result.add(validation.ShouldBeInNamespacesError("cluster", fake.RoleAtPath("cluster/role.yaml")))
	result.add(validation.ShouldBeInClusterError("namespaces", fake.ClusterRoleAtPath("namespaces/clusterrole.yaml")))

	// 1040
	result.add(hierarchyconfig.UnknownResourceInHierarchyConfigError(hierarchyconfig.FileGroupKindHierarchyConfig{
		GK:            kinds.Anvil().GroupKind(),
		HierarchyMode: v1.HierarchyModeDefault,
		Resource:      fake.HierarchyConfig(),
	}))

	// 1041
	result.add(hierarchyconfig.UnsupportedResourceInHierarchyConfigError(hierarchyconfig.FileGroupKindHierarchyConfig{
		GK:            kinds.Namespace().GroupKind(),
		HierarchyMode: v1.HierarchyModeDefault,
		Resource:      fake.HierarchyConfig(),
	}))

	// 1042
	result.add(hierarchyconfig.IllegalHierarchyModeError(hierarchyconfig.FileGroupKindHierarchyConfig{
		GK:            kinds.Role().GroupKind(),
		HierarchyMode: "invalid",
		Resource:      fake.HierarchyConfig(),
	}, "invalid"))

	// 1043
	result.add(nonhierarchical.UnsupportedObjectError(fake.CustomResourceDefinition()))

	// 1044
	result.add(semantic.UnsyncableResourcesInLeaf(node("namespaces/foo")))
	result.add(semantic.UnsyncableResourcesInNonLeaf(node("namespaces/foo")))

	// 1045
	result.add(syntax.IllegalFieldsInConfigError(fake.Role(), id.Status))

	// 1046
	result.add(hierarchyconfig.ClusterScopedResourceInHierarchyConfigError(hierarchyconfig.FileGroupKindHierarchyConfig{
		GK:            kinds.ClusterRole().GroupKind(),
		HierarchyMode: v1.HierarchyModeDefault,
		Resource:      fake.HierarchyConfig(),
	}))

	// 1047
	result.add(semantic.UnsupportedCRDRemovalError(fake.CustomResourceDefinition()))

	// 1048
	result.add(nonhierarchical.InvalidCRDNameError(fake.CustomResourceDefinition()))

	// 1049 is Deprecated.

	// 1050
	result.add(syntax.DeprecatedGroupKindError(
		asttesting.NewFakeFileObject(schema.GroupVersionKind{
			Group:   "extensions",
			Version: "v1beta1",
			Kind:    kinds.Deployment().Kind,
		}, "namespaces/deployment.yaml"), kinds.Deployment()))

	// 1051 is Deprecated.

	// 1052
	result.add(nonhierarchical.IllegalNamespaceOnClusterScopedResourceError(fake.ClusterRole(core.Namespace("foo"))))

	// 1053
	result.add(nonhierarchical.MissingNamespaceOnNamespacedResourceError(fake.Role(core.Namespace(""))))

	// 1054
	result.add(filesystem.InvalidAnnotationValueError(fake.Role(), []string{"foo", "bar"}))

	// 2001
	result.add(status.PathWrapError(errors.New("error creating directory"), "namespaces/foo"))

	// 2002
	result.add(status.APIServerError(errors.New("problem talking to Kubernetes cluster"), "could not create connection"))

	// 2003
	result.add(status.OSWrap(errors.New("problem reading file")))

	// 2004
	result.add(status.SourceError.Sprint("unable to connect to Git repository").Build())

	// 2010
	result.add(status.ResourceWrap(errors.New("specific problem with resource"), "general message", fake.Role()))

	// 2011
	result.add(status.MissingResourceWrap(errors.New("the Role 'foo' in Namespace 'bar' was not found"),
		"unable to update resource", fake.Role(core.Name("foo"), core.Namespace("bar"))))

	// 2012
	result.add(status.MultipleSingletonsError(fake.Repo(), fake.Repo()))

	// 9999
	result.add(status.UndocumentedError("error not yet documented"))

	return result
}

// add adds example errors for a specific error code for use in documentation.
func (e *exampleErrors) add(err status.Error) {
	// Ensures example error can be displayed.
	_ = err.Error()
	code := err.Code()
	(*e)[code] = append((*e)[code], err)
}

type path string

var _ id.Path = path("")

// SlashPath implements id.Path
func (p path) SlashPath() string {
	return string(p)
}

// OSPath implements id.Path
func (p path) OSPath() string {
	return string(p)
}

func node(s string) treeNode {
	splits := strings.Split(s, "/")
	name := splits[len(splits)-1]
	return treeNode{path: path(s), name: name}
}

type treeNode struct {
	path
	name string
}

var _ id.TreeNode = treeNode{}

// Name implements id.TreeNode
func (n treeNode) Name() string {
	return n.name
}
