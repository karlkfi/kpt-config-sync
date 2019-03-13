package importer

import (
	"os"
	"path/filepath"

	"github.com/google/nomos/cmd/nomos/flags"
	"github.com/google/nomos/pkg/api/policyhierarchy"
	"github.com/google/nomos/pkg/api/policyhierarchy/v1"
	"github.com/google/nomos/pkg/client/restconfig"
	"github.com/google/nomos/pkg/cloner"
	"github.com/google/nomos/pkg/cloner/filter"
	"github.com/google/nomos/pkg/cloner/mutate"
	"github.com/google/nomos/pkg/kinds"
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

// Cmd exports resources in the current kubectl context into the specified directory.
var Cmd = &cobra.Command{
	Use: "import",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Allow other outputs than os.Stderr.
		errOutput := cloner.NewStandardErrorOutput()
		dir, err := filepath.Abs(flags.Path.String())
		errOutput.AddAndDie(errors.Wrap(err, "failed to get absolute path"))

		clientConfig, err := restconfig.NewClientConfig()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get kubectl config"))

		restConfig, err := clientConfig.ClientConfig()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get rest.Config"))

		// TODO(119066037): Override the host in a way that doesn't involve overwriting defaults set internally in client-go.
		clientcmd.ClusterDefaults = clientcmdapi.Cluster{Server: restConfig.Host}

		factory := cmdutil.NewFactory(&genericclioptions.ConfigFlags{})

		discoveryClient, err := factory.ToDiscoveryClient()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get discovery client"))

		apiResources := cloner.ListResources(discoveryClient, errOutput)
		errOutput.DieIfPrintedErrors("failed to list available API objects")

		dynamicClient, err := factory.DynamicClient()
		errOutput.AddAndDie(errors.Wrap(err, "failed to get dynamic client"))

		lister := cloner.NewResourceLister(cloner.DynamicResourcer{Interface: dynamicClient})

		var objects []ast.FileObject
		for _, apiResource := range apiResources {
			gk := schema.GroupKind{Group: apiResource.Group, Kind: apiResource.Kind}
			if ignoredGroupKinds[gk] {
				continue
			}
			resources := lister.List(apiResource, errOutput)
			objects = append(objects, resources...)
		}

		objects = filter.Objects(objects, filter.Any(
			ignoreSystemNameGroups,
			ignoreSystemNamespaces,
			ignoreKubernetesSystemLabels,
			ignoreCriticalPriorityClasses,
		))

		mutate.ApplyAll(objects,
			mutate.Unapply(),
			removeNomosLables,
			removeNomosAnnotations,
			removeAppliedConfig,
		)

		pather := cloner.NewPather(apiResources...)
		pather.AddPaths(objects)

		printer := &printers.YAMLPrinter{}
		for _, object := range objects {
			err2 := writeObject(printer, dir, object)
			errOutput.Add(err2)
		}
		errOutput.DieIfPrintedErrors("encountered errors writing resources to files")
	},
}

func writeObject(printer printers.ResourcePrinter, dir string, object ast.FileObject) error {
	if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(object.Dir().RelativeSlashPath())), 0750); err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(dir, filepath.FromSlash(object.RelativeSlashPath())))
	if err != nil {
		return err
	}

	return printer.PrintObj(object.Object, file)
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
	// Nodes represent literal worker machine.
	schema.GroupKind{Kind: "Node"}: true,
	// NodeMetrics keep track of the transient state of machines in the cluster.
	schema.GroupKind{Kind: "NodeMetrics", Group: "metrics.k8s.io"}: true,
	// PodMetrics keeps track of the transient state of pods in the cluster.
	schema.GroupKind{Kind: "PodMetrics", Group: "metrics.k8s.io"}: true,
	// Secrets hold data that shouldn't be shared by default.
	schema.GroupKind{Kind: "Secret"}: true,
	// ClusterPolicy is an internal Nomos type we don't support syncing.
	kinds.ClusterPolicy().GroupKind(): true,
	// PolicyNode is an internal Nomos type we don't support syncing.
	kinds.PolicyNode().GroupKind(): true,
	// Sync is an internal Nomos type we don't support syncing.
	kinds.Sync().GroupKind(): true,
}

// ignoreSystemNameGroups ignores resources in name groups indicating they are a critical part of
// Kubernetes or Nomos functionality.
var ignoreSystemNameGroups = filter.Any(
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
var ignoreSystemNamespaces = filter.Any(
	filter.Namespace("default"),
	filter.Namespace("kube-public"),
	filter.Namespace("kube-system"),
	filter.Namespace(policyhierarchy.ControllerNamespace),
)

// ignoreKubernetesSystemLabels returns true for resources which have Kubernetes system labels
// set.
var ignoreKubernetesSystemLabels = filter.Any(
	// addonmanager.kubernetes.io/mode indicates the resource is managed by an addon.
	filter.Label("addonmanager.kubernetes.io/mode"),
	//config.gke.io/system indicates the resource is part of the Nomos installation
	filter.Label("config.gke.io/system"),
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
var ignoreCriticalPriorityClasses = filter.All(
	filter.GroupKind(schema.GroupKind{Group: "scheduling.k8s.io", Kind: "PriorityClass"}),
	filter.Any(filter.Name("system-cluster-critical"), filter.Name("system-node-critical")),
)

// removeNomosLables removes all Nomos labels.
var removeNomosLables = mutate.RemoveLabelGroup("config.gke.io")

// removeNomosAnnotations removes non-input Nomos annotations.
var removeNomosAnnotations = mutate.RemoveAnnotationGroup(v1.NomosPrefix)

// removeAppliedConfig removes the annotation holding a JSON representation of the last call to
// kubectl apply on the resource.
var removeAppliedConfig = mutate.RemoveAnnotation(mutate.AppliedConfiguration)
