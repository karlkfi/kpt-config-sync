package examples

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/api/configsync"
	"github.com/google/nomos/pkg/applier"
	"github.com/google/nomos/pkg/core"
	"github.com/google/nomos/pkg/declared"
	"github.com/google/nomos/pkg/importer/analyzer/hnc"
	"github.com/google/nomos/pkg/importer/analyzer/transform/selectors"
	"github.com/google/nomos/pkg/importer/analyzer/validation"
	"github.com/google/nomos/pkg/importer/analyzer/validation/hierarchyconfig"
	"github.com/google/nomos/pkg/importer/analyzer/validation/metadata"
	"github.com/google/nomos/pkg/importer/analyzer/validation/nonhierarchical"
	"github.com/google/nomos/pkg/importer/analyzer/validation/semantic"
	"github.com/google/nomos/pkg/importer/analyzer/validation/syntax"
	"github.com/google/nomos/pkg/importer/analyzer/validation/system"
	"github.com/google/nomos/pkg/importer/filesystem/cmpath"
	"github.com/google/nomos/pkg/importer/id"
	"github.com/google/nomos/pkg/importer/reader"
	"github.com/google/nomos/pkg/kinds"
	csmetadata "github.com/google/nomos/pkg/metadata"
	"github.com/google/nomos/pkg/parse"
	"github.com/google/nomos/pkg/status"
	"github.com/google/nomos/pkg/syncer/client"
	"github.com/google/nomos/pkg/syncer/reconcile"
	"github.com/google/nomos/pkg/testing/fake"
	"github.com/google/nomos/pkg/util/clusterconfig"
	"github.com/google/nomos/pkg/vet"
	"github.com/google/nomos/pkg/webhook/configuration"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ExamplesOrDeprecated contains either a list of example errors, or that the
// id is deprecated.
type ExamplesOrDeprecated struct {
	Examples   []status.Error
	Deprecated bool
}

// AllExamples is a map from error codes to either example errors, or a mark that
// the code is deprecated.
type AllExamples map[string]ExamplesOrDeprecated

// Generate generates example errors for documentation.
// KNV1XXX means the user has a mistake in their repository they need to fix.
// KNV2XXX means something went wrong in the cluster - it could be transient or users may need to change something on the cluster.
// KNV9XXX means we made a mistake programming, and users should file a bug.
// More details in go/nomos-style.
func Generate() AllExamples {
	// exampleErrors is a map of exampleErrors of each error type. For documentation purposes, i.e. for use
	// in the internal-only nomoserrors command.
	result := make(AllExamples)

	// 1000
	result.markDeprecated("1000")

	// 1001 is Deprecated.
	result.markDeprecated("1001")

	// 1002 is Deprecated.
	result.markDeprecated("1002")

	// 1003
	result.add(validation.IllegalNamespaceSubdirectoryError(node("namespaces/foo/bar"), node("namespaces/foo")))

	// 1004
	result.add(nonhierarchical.IllegalNamespaceSelectorAnnotationError(fake.Namespace("namespaces/foo")))
	result.add(nonhierarchical.IllegalClusterSelectorAnnotationError(fake.Cluster(), csmetadata.ClusterNameSelectorAnnotationKey))

	// 1005
	result.add(nonhierarchical.IllegalManagementAnnotationError(fake.Role(), "invalid"))

	// 1006
	result.add(status.ObjectParseError(fake.Role(), errors.New("wrong type")))

	// 1007
	result.add(validation.IllegalAbstractNamespaceObjectKindError(fake.RoleAtPath("namespaces/foo/bar/role.yaml")))

	// 1008 is Deprecated.
	result.markDeprecated("1008")

	// 1009
	result.add(metadata.IllegalMetadataNamespaceDeclarationError(
		fake.RoleAtPath("namespaces/foo/r.yaml", core.Namespace("bar")), "foo"))

	// 1010
	result.add(metadata.IllegalAnnotationDefinitionError(fake.Role(), []string{csmetadata.ConfigManagementPrefix + "illegal-annotation"}))

	// 1011
	result.add(metadata.IllegalLabelDefinitionError(fake.Role(), []string{csmetadata.ConfigManagementPrefix + "label"}))

	// 1012 is Deprecated.
	result.markDeprecated("1012")

	// 1013
	result.add(selectors.ObjectHasUnknownClusterSelector(fake.Role(), "undeclared-selector"))
	result.add(selectors.ObjectHasUnknownNamespaceSelector(fake.Role(), "undeclared-selector"))
	result.add(selectors.ObjectNotInNamespaceSelectorSubdirectory(
		fake.RoleAtPath("namespaces/foo/role.yaml"),
		fake.NamespaceSelectorAtPathWithName("namespaces/bar/selector.yaml", "default-ns-selector")))

	// 1014
	result.add(selectors.InvalidSelectorError(fake.NamespaceSelector(), errors.New("some parse error")))
	result.add(selectors.EmptySelectorError(fake.NamespaceSelector()))

	// 1015 is Deprecated.
	result.markDeprecated("1015")

	// 1016 is Deprecated.
	result.markDeprecated("1016")

	// 1017
	result.add(system.MissingRepoError())

	// 1018 is Deprecated.
	result.markDeprecated("1018")

	// 1019
	result.add(metadata.IllegalTopLevelNamespaceError(fake.Namespace("namespaces")))

	// 1020
	result.add(metadata.InvalidNamespaceNameError(fake.Namespace("namespaces/foo", core.Name("bar")), "foo"))

	// 1021
	result.add(status.UnknownObjectKindError(fake.UnstructuredAtPath(schema.GroupVersionKind{
		Group:   "com.me",
		Version: "v1",
		Kind:    "Engineer",
	}, "namespaces/foo/engineer.yaml")))

	// 1022 is Deprecated.
	result.markDeprecated("1022")

	// 1023 is Deprecated.
	result.markDeprecated("1023")

	// 1024 is Deprecated.
	result.markDeprecated("1024")

	// 1025 is Deprecated.
	result.markDeprecated("1025")

	// 1026 is Deprecated.
	result.markDeprecated("1026")

	// 1027
	result.add(system.UnsupportedRepoSpecVersion(fake.Repo(fake.RepoVersion("")), "0.0.0"))

	// 1028
	result.add(syntax.ReservedDirectoryNameError(cmpath.RelativeSlash("namespaces/" + configmanagement.ControllerNamespace)))
	result.add(syntax.InvalidDirectoryNameError(cmpath.RelativeSlash("namespaces/ABC")))

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
	result.add(syntax.IllegalSystemResourcePlacementError(fake.HierarchyConfigAtPath("system/hierarchy-config.yaml")))

	// 1034
	result.add(nonhierarchical.IllegalNamespace(fake.Namespace("namespaces/" + configmanagement.ControllerNamespace)))
	result.add(nonhierarchical.ObjectInIllegalNamespace(fake.RoleAtPath("namespaces/"+configmanagement.ControllerNamespace+"/role.yaml",
		core.Namespace("namespaces/"+configmanagement.ControllerNamespace))))

	// 1035 is Deprecated.
	result.markDeprecated("1035")

	// 1036
	result.add(nonhierarchical.InvalidMetadataNameError(fake.Role(core.Name("ABC"))))

	// 1037 is Deprecated.
	result.markDeprecated("1037")

	// 1038
	result.add(syntax.IllegalKindInNamespacesError(fake.NamespaceSelectorAtPath("namespaces/foo/ns-selector.yaml")))

	// 1039
	result.add(validation.ShouldBeInSystemError(fake.RepoAtPath("namespaces/repo.yaml")))
	result.add(validation.ShouldBeInClusterRegistryError(fake.ClusterAtPath("namespaces/cluster.yaml")))
	result.add(validation.ShouldBeInClusterError(fake.ClusterRoleAtPath("namespaces/clusterrole.yaml")))
	result.add(validation.ShouldBeInNamespacesError(fake.RoleAtPath("cluster/role.yaml")))

	// 1040 is Deprecated.
	result.markDeprecated("1040")

	// 1041
	result.add(hierarchyconfig.UnsupportedResourceInHierarchyConfigError(fake.HierarchyConfig(), kinds.Namespace().GroupKind()))

	// 1042
	result.add(hierarchyconfig.IllegalHierarchyModeError(fake.HierarchyConfig(), kinds.Role().GroupKind(), "invalid"))

	// 1043
	result.add(nonhierarchical.UnsupportedObjectError(fake.CustomResourceDefinitionV1Beta1()))
	result.add(nonhierarchical.UnsupportedObjectError(fake.CustomResourceDefinitionV1()))

	// 1044
	result.add(semantic.UnsyncableResourcesInLeaf(node("namespaces/foo")))
	result.add(semantic.UnsyncableResourcesInNonLeaf(node("namespaces/foo")))

	// 1045
	result.add(syntax.IllegalFieldsInConfigError(fake.Role(), id.Status))

	// 1046
	result.add(hierarchyconfig.ClusterScopedResourceInHierarchyConfigError(fake.HierarchyConfig(), kinds.ClusterRole().GroupKind()))

	// 1047
	result.add(nonhierarchical.UnsupportedCRDRemovalError(fake.CustomResourceDefinitionV1Beta1()))

	// 1048
	result.add(nonhierarchical.InvalidCRDNameError(fake.CustomResourceDefinitionV1Beta1(), "default-names.apiextensions.k8s.io"))

	// 1049 is Deprecated.
	result.markDeprecated("1049")

	// 1050
	result.add(nonhierarchical.DeprecatedGroupKindError(
		fake.UnstructuredAtPath(schema.GroupVersionKind{
			Group:   "extensions",
			Version: "v1beta1",
			Kind:    kinds.Deployment().Kind,
		}, "namespaces/deployment.yaml"), kinds.Deployment()))

	// 1051 is Deprecated.
	result.markDeprecated("1051")

	// 1052
	result.add(nonhierarchical.IllegalNamespaceOnClusterScopedResourceError(fake.ClusterRole(core.Namespace("foo"))))

	// 1053
	result.add(nonhierarchical.MissingNamespaceOnNamespacedResourceError(fake.Role(core.Namespace(""))))

	// 1054
	result.add(reader.InvalidAnnotationValueError(fake.Role(), []string{"foo", "bar"}))

	// 1055
	result.add(nonhierarchical.InvalidNamespaceError(fake.Repo(core.Namespace("FOO"))))

	// 1056
	result.add(nonhierarchical.ManagedResourceInUnmanagedNamespace("foo", fake.Role()))

	// 1057
	result.add(hnc.IllegalDepthLabelError(fake.Role(), []string{"label" + csmetadata.DepthSuffix}))

	// 1058
	result.add(parse.BadScopeErr(fake.Role(core.Namespace("shipping")), "dev"))

	// 1059 is Deprecated.
	result.markDeprecated("1059")

	// 1060
	result.add(applier.ManagementConflictError(fake.Role()))

	// 1061
	result.add(nonhierarchical.InvalidSyncName(configsync.RepoSyncName, fake.RepoSyncObject(core.Name("invalid"))))

	// 1062 is Deprecated.
	result.markDeprecated("1062")

	// 1063 is Deprecated.
	result.markDeprecated("1063")

	// 1064
	p, _ := cmpath.AbsoluteSlash("/api-resources.txt")
	result.add(vet.InvalidScopeValue(p, "rbac      other     Role", "other"))
	result.add(vet.UnableToReadAPIResources(p, errors.New("missing file permissions")))
	result.add(vet.MissingAPIGroup(p))

	// 1065
	result.add(clusterconfig.MalformedCRDError(
		fmt.Errorf("spec.names.shortNames accessor error: foo is of the type string, expected []interface{}"),
		fake.CustomResourceDefinitionV1Object()))

	// 1066
	result.add(selectors.ClusterSelectorAnnotationConflictError(fake.NamespaceObject("my-namespace")))

	// 1067
	result.add(status.EncodeDeclaredFieldError(fake.NamespaceObject("my-namespace"),
		fmt.Errorf(".spec.version not defined")))

	// 2001
	result.add(status.PathWrapError(errors.New("error creating directory"), "namespaces/foo"))

	// 2002
	result.add(status.APIServerError(errors.New("problem talking to Kubernetes cluster"), "could not create connection"))

	// 2003
	result.add(status.OSWrap(errors.New("problem reading file")))

	// 2004
	result.add(status.SourceError.Sprint("unable to connect to Git repository").Build())

	// 2005
	result.add(reconcile.FightWarning(9.5, fake.NamespaceObject("gatekeeper-system")))

	// 2006
	result.add(status.EmptySourceError(10, "namespaces"))
	result.add(declared.DeleteAllNamespacesError([]string{"shipping", "billing"}))

	// 2007 is Deprecated.
	result.markDeprecated("2007")

	// 2008
	result.add(client.ConflictCreateAlreadyExists(errors.New("already exists"), fake.RoleObject()))
	result.add(client.ConflictUpdateOldVersion(errors.New("old version"), fake.RoleObject()))
	result.add(client.ConflictUpdateDoesNotExist(errors.New("does not exist"), fake.RoleObject()))

	// 2009
	result.add(applier.Error(errors.New("failed to initialize an error")))

	// 2010
	result.add(status.ResourceWrap(errors.New("specific problem with resource"), "general message", fake.Role()))

	// 2011
	result.add(status.MissingResourceWrap(errors.New("the Role 'foo' in Namespace 'bar' was not found"),
		"unable to update resource", fake.Role(core.Name("foo"), core.Namespace("bar"))))

	// 2012
	result.add(status.MultipleSingletonsError(fake.Repo(), fake.Repo()))

	// 2013
	result.add(status.InsufficientPermissionErrorBuilder.Sprint("could not create resources").Wrap(
		errors.New("deployments.apps is forbidden: User 'Bob' cannot create resources")).Build())

	// 2014
	result.add(configuration.InvalidWebhookWarning("invalid webhook"))

	// 2015
	result.add(status.HydrationError.Sprint("Rendering error").Build())

	// 9997
	result.add(status.HydrationInProgress("abcd123"))

	// 9998
	result.add(status.InternalError("we made a mistake"))

	// 9999
	result.add(status.UndocumentedError("error not yet documented"))

	return result
}

// Add adds the given error to the collection examples of errors.
func (e *ExamplesOrDeprecated) Add(error status.Error) {
	e.Examples = append(e.Examples, error)
}

// add adds example errors for a specific error code for use in documentation.
func (e AllExamples) add(err status.Error) {
	// Ensures example error can be displayed.
	_ = err.Error()
	code := err.Code()
	examples := e[code]
	examples.Add(err)
	e[code] = examples
}

func (e AllExamples) markDeprecated(id string) {
	e[id] = ExamplesOrDeprecated{
		Examples:   nil,
		Deprecated: true,
	}
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
