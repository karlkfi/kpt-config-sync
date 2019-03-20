package importer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/importer"
	"github.com/google/nomos/pkg/importer/filter"
	"github.com/google/nomos/pkg/importer/mutate"
	"github.com/google/nomos/pkg/kinds"
	"github.com/google/nomos/pkg/object"
	"github.com/google/nomos/pkg/policyimporter/analyzer/ast"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericclioptions/printers"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var force bool
var silent bool

func init() {
	Cmd.Flags().BoolVar(&silent, "silent", false, "Only print errors")
	Cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files")
}

// Cmd exports resources in the current kubectl context into the specified directory.
var Cmd = &cobra.Command{
	Use:   "import",
	Short: `Downloads all resources from the current kubectl context and formats them into a valid Config Management repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		infoOut := importer.NewStdOutput()
		if silent {
			infoOut = importer.NewNilOutput()
		}

		// TODO: Allow other outputs than os.Stderr.
		errOutput := importer.NewStandardErrorOutput()
		dir, err := filepath.Abs(flags.Path.String())
		errOutput.AddAndDie(errors.Wrap(err, "failed to get absolute path"))

		clientConfig, err := restconfig.NewClientConfig()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get kubectl config"))

		restConfig, err := clientConfig.ClientConfig()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get rest.Config"))

		// TODO(119066037): Override the host in a way that doesn't involve overwriting defaults set internally in client-go.
		clientcmd.ClusterDefaults = clientcmdapi.Cluster{Server: restConfig.Host}

		rawConfig, err := clientConfig.RawConfig()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get raw config"))
		infoOut.Printfln("\n*** Importing resources from context %s ***\n", rawConfig.CurrentContext)

		factory := cmdutil.NewFactory(&genericclioptions.ConfigFlags{})

		discoveryClient, err := factory.ToDiscoveryClient()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get discovery client"))

		infoOut.Printfln("Listing available APIResources")
		apiResources := importer.ListResources(discoveryClient, errOutput)
		errOutput.DieIfPrintedErrors("failed to list available API objects")

		dynamicClient, err := factory.DynamicClient()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get dynamic client"))

		lister := importer.NewResourceLister(importer.DynamicResourcer{Interface: dynamicClient})

		infoOut.Printfln("Listing resources for each APIResource")
		var objects []ast.FileObject
		for _, apiResource := range apiResources {
			gk := schema.GroupKind{Group: apiResource.Group, Kind: apiResource.Kind}
			if ignoredGroupKinds[gk] {
				infoOut.Printfln("  Ignoring %s", gk.String())
				continue
			}
			resources := lister.List(apiResource, errOutput)
			objects = append(objects, resources...)
		}

		infoOut.Printfln("Filtering out system resources")
		objects = object.Filter(objects, object.Any(
			ignoreSystemNameGroups,
			ignoreSystemNamespaces(infoOut),
			ignoreKubernetesSystemLabels,
			ignoreCriticalPriorityClasses,
		))

		object.Mutate(
			mutate.Unapply(infoOut),
			removeNomosLables,
			removeNomosAnnotations,
			removeAppliedConfig,
			cleanNamespaces,
			exportObjectMeta,
			mutate.Prune(),
		).Apply(objects)

		pather := importer.NewPather(apiResources...)
		pather.AddPaths(objects)

		infoOut.Printfln("Writing %d resources to disk", len(objects))
		printer := &printers.YAMLPrinter{}
		for _, object := range objects {
			if _, err := os.Stat(object.OSPath()); os.IsNotExist(err) {
				// We want this; do nothing.
			} else if err != nil {
				errOutput.AddAndDie(err)
			} else {
				if !force {
					fmt.Printf("  Import would overwrite existing file %s\n  Use --force to proceed\n", object.OSPath())
					os.Exit(1)
				}
			}
		}
		for _, object := range objects {
			err2 := util.WriteObject(printer, dir, object)
			errOutput.Add(err2)
		}
		errOutput.DieIfPrintedErrors("encountered errors writing resources to files")
		infoOut.Printfln("Done")
	},
}

var ignoredGroupKinds = map[schema.GroupKind]bool{
	// CertificateSigningRequests are transient resources.
	schema.GroupKind{Kind: "CertificateSigningRequest", Group: "certificates.k8s.io"}: true,
	// ComponentStatus is managed by Kubernetes and changes constantly.
	schema.GroupKind{Kind: "ComponentStatus"}: true,
	// ComponentStatus is an immutable snapshot of controller data.
	schema.GroupKind{Kind: "ControllerRevision", Group: "apps"}: true,
	// CustomResourceDefinitions are not yet supported for syncing.
	schema.GroupKind{Kind: "CustomResourceDefinition", Group: "apiextensions.k8s.io"}: true,
	// Events are transient things that happened on the cluster and shouldn't be synced.
	schema.GroupKind{Kind: "Event"}: true,
	// Nodes represent literal worker machine.
	schema.GroupKind{Kind: "Node"}: true,
	// NodeMetrics keep track of the transient state of machines in the cluster.
	schema.GroupKind{Kind: "NodeMetrics", Group: "metrics.k8s.io"}: true,
	// PodMetrics keeps track of the transient state of pods in the cluster.
	schema.GroupKind{Kind: "PodMetrics", Group: "metrics.k8s.io"}: true,
	// Secrets hold data that shouldn't be shared by default.
	schema.GroupKind{Kind: "Secret"}: true,
	// ClusterConfig is an internal Nomos type we don't support syncing.
	kinds.ClusterConfig().GroupKind(): true,
	// NamespaceConfig is an internal Nomos type we don't support syncing.
	kinds.NamespaceConfig().GroupKind(): true,
	// Sync is an internal Nomos type we don't support syncing.
	kinds.Sync().GroupKind(): true,
	// HierarchicalQuota is not something users should create directly.
	kinds.HierarchicalQuota().GroupKind(): true,
}

// ignoreSystemNameGroups ignores resources in name groups indicating they are a critical part of
// Kubernetes or Nomos functionality.
var ignoreSystemNameGroups = object.Any(
	// system: resources are part of the Kubernetes system.
	filter.NameGroup("system"),
	// gce: resources are part of Google Compute Engine.
	filter.NameGroup("gce"),
	// metrics-server: resources are background processes collecing metrics on resource usage.
	filter.NameGroup("metrics-server"),
	// configmanagement.gke.io: resources are part of the Nomos installation.
	filter.NameGroup(policyhierarchy.GroupName),
)

// ignoreSystemNamespaces ignores all of the Namespaces which have internal Kubernetes and Nomos
// resources. We don't support syncing any of these namespaces.
func ignoreSystemNamespaces(out importer.InfoOutput) object.Predicate {
	ignoredNamespaces := []string{"default", "kube-public", "kube-system", policyhierarchy.ControllerNamespace}
	var namespaceFilters []object.Predicate
	for _, n := range ignoredNamespaces {
		out.Printfln("  Ignoring %s Namespace", n)
		namespaceFilters = append(namespaceFilters, filter.Namespace(n))
	}

	return object.Any(namespaceFilters...)
}

// ignoreKubernetesSystemLabels returns true for resources which have Kubernetes system labels
// set.
var ignoreKubernetesSystemLabels = object.Any(
	// addonmanager.kubernetes.io/mode indicates the resource is managed by an addon.
	filter.Label("addonmanager.kubernetes.io/mode"),
	//config.gke.io/system indicates the resource is part of the Nomos installation
	filter.Label(policyhierarchy.GroupName+"/system"),
	// k8s-app indicates the resource is part of a Kubernetes app.
	filter.Label("k8s-app"),
	// kube-aggregator.kubernetes.io/automanaged indicates the resource is automatically managed by Kubernetes.
	filter.Label("kube-aggregator.kubernetes.io/automanaged"),
	// kubernetes.io/bootstrapping indicates the resource was automatically generated on Kubernetes installation.
	filter.Label("kubernetes.io/bootstrapping"),
	// kubernetes.io/cluster-service are Kubernetes cluster service resources.
	filter.Label("kubernetes.io/cluster-service"),
)

// ignoreCriticalPriorityClasses returns false for resources which are the default critical
// priority classes, which are essential to clusters and nodes functioning. Modifying these can
// cause processes critical to cluster functioning to get preempted.
var ignoreCriticalPriorityClasses = object.All(
	filter.GroupKind(schema.GroupKind{Group: "scheduling.k8s.io", Kind: "PriorityClass"}),
	object.Any(filter.Name("system-cluster-critical"), filter.Name("system-node-critical")),
)

// removeNomosLables removes all Nomos labels.
var removeNomosLables = mutate.RemoveLabelGroup(policyhierarchy.GroupName)

// removeNomosAnnotations removes non-input Nomos annotations.
var removeNomosAnnotations = mutate.RemoveAnnotationGroup(policyhierarchy.GroupName)

// removeAppliedConfig removes the annotation holding a JSON representation of the last call to
// kubectl apply on the resource.
var removeAppliedConfig = mutate.RemoveAnnotation(mutate.AppliedConfiguration)

// cleanNamespaces removes the kubernetes finalizer and the status.phase from Namespaces.
var cleanNamespaces = object.Mutate(
	// Kubernetes manages this finalizer on Namespaces.
	mutate.Remove(mutate.Key("spec", "finalizers").Value("kubernetes")),
	// transient Namespace state managed by Kubernetes
	mutate.Remove(mutate.Key("status", "phase")),
).If(filter.GroupKind(kinds.Namespace().GroupKind()))

// exportObjectMeta mimics the behavior of exportObjectMeta() from
// kubernetes/staging/src/k8s.io/apiserver/pkg/registry/generic/registry/store.go.
// Simply setting them to empty string in the meta object doesn't remove them; we have to directly
// modify the underlying Unstructured.
var exportObjectMeta = object.Mutate(
	// creationTimestamp tracks when an object was creates in Kubernetes, and shouldn't be managed by Nomos.
	mutate.Remove(mutate.Key("metadata", "creationTimestamp")),
	// deletionTimestamp tracks when the request to delete the resources was received, and shouldn't be managed in version control.
	mutate.Remove(mutate.Key("metadata", "deletionTimestamp")),
	// namespace is valid on resources, but we automatically infer it so it is better to not declare the field.
	mutate.Remove(mutate.Key("metadata", "namespace")),
	// resourceVersion is automatically generated and managed by Kubernetes.
	mutate.Remove(mutate.Key("metadata", "resourceVersion")),
	// selfLink is automatically generated and managed by Kubernetes.
	mutate.Remove(mutate.Key("metadata", "selfLink")),
	// uid is automatically generated and managed by Kubernetes.
	mutate.Remove(mutate.Key("metadata", "uid")),
)
