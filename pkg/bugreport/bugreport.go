package bugreport

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/golang/glog"
	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	"github.com/google/nomos/pkg/api/configsync/v1alpha1"
	"github.com/google/nomos/pkg/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/nomos/cmd/nomos/status"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/cmd/nomos/version"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/client/restconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	corev1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type coreClient interface {
	CoreV1() corev1Client.CoreV1Interface
}

func operatorLabelSelectorOrDie() labels.Requirement {
	ret, err := labels.NewRequirement("k8s-app", selection.Equals, []string{"config-management-operator"})
	if err != nil {
		panic(err)
	}
	return *ret
}

// Filepath for bugreport directory
const (
	Namespace    = "namespaces"
	ClusterScope = "cluster"
	Raw          = "raw"
	Processed    = "processed"
)

// BugReporter handles basic data gathering tasks for generating a
// bug report
type BugReporter struct {
	client    client.Reader
	clientSet *kubernetes.Clientset
	cm        *unstructured.Unstructured
	enabled   map[Product]bool
	util.ConfigManagementClient
	k8sContext string
	// report file
	outFile       *os.File
	writer        *zip.Writer
	name          string
	ErrorList     []error
	WritingErrors []error
}

// New creates a new BugReport
func New(ctx context.Context, cfg *rest.Config) (*BugReporter, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	cm := &unstructured.Unstructured{}
	cm.SetGroupVersionKind(schema.GroupVersionKind{
		Group: configmanagement.GroupName,
		// we rely on a specific version implicitly later in the code, so this should
		// be hardcoded
		Version: "v1",
		Kind:    configmanagement.OperatorKind,
	})
	errorList := []error{}
	currentk8sContext, err := restconfig.CurrentContextName()
	if err != nil {
		errorList = append(errorList, err)
	}

	if err := c.Get(ctx, types.NamespacedName{Name: util.ConfigManagementName}, cm); err != nil {
		if meta.IsNoMatchError(err) {
			fmt.Println("kind <<" + configmanagement.OperatorKind + ">> is not registered with the cluster")
		} else if errors.IsNotFound(err) {
			fmt.Println("ConfigManagement object not found")
		} else {
			errorList = append(errorList, err)
		}
	}

	return &BugReporter{
		client:        c,
		clientSet:     cs,
		cm:            cm,
		k8sContext:    currentk8sContext,
		ErrorList:     errorList,
		WritingErrors: []error{},
	}, nil
}

// EnabledServices returns the set of services that are enabled
func (b *BugReporter) EnabledServices() map[Product]bool {
	if b.enabled == nil {
		enabled := make(map[Product]bool)

		// We can safely ignore errors here, because if this request doesn't succeed,
		// Policy Controller is not enabled
		enabled[PolicyController], _, _ = unstructured.NestedBool(b.cm.Object, "spec", "policyController", "enabled")
		// Same for KCC
		enabled[KCC], _, _ = unstructured.NestedBool(b.cm.Object, "spec", "configConnector", "enabled")
		// Same for Config Sync, though here the "disabled" condition is if enableMultiRepo is true or if the git
		// config is "empty", which involves looking for an empty proxy config
		configSyncEnabled := false
		enableMultiRepo, _, _ := unstructured.NestedBool(b.cm.Object, "spec", "enableMultiRepo")
		if enableMultiRepo {
			configSyncEnabled = true
		} else {
			syncGitCfg, _, _ := unstructured.NestedMap(b.cm.Object, "spec", "git")
			for k := range syncGitCfg {
				if k != "proxy" {
					configSyncEnabled = true
				}
			}
			proxy, _, _ := unstructured.NestedMap(syncGitCfg, "proxy")
			if len(proxy) > 0 {
				configSyncEnabled = true
			}
		}

		enabled[ConfigSync] = configSyncEnabled
		enabled[ResourceGroup] = enableMultiRepo
		enabled[ConfigSyncMonitoring] = true
		b.enabled = enabled
	}

	return b.enabled
}

// FetchLogSources provides a set of Readables for all of nomos' container logs
// TODO: Still need to figure out a good way to test this
func (b *BugReporter) FetchLogSources(ctx context.Context) []Readable {
	var toBeLogged logSources

	// for each namespace, generate a list of logSources
	listOps := client.ListOptions{LabelSelector: labels.NewSelector().Add(operatorLabelSelectorOrDie())}
	sources, err := b.logSourcesForNamespace(ctx, metav1.NamespaceSystem, listOps, nil)
	if err != nil {
		b.ErrorList = append(b.ErrorList, err)
	} else {
		toBeLogged = append(toBeLogged, sources...)
	}

	listOps = client.ListOptions{}
	nsLabels := map[string]string{"configmanagement.gke.io/configmanagement": "config-management"}
	productAndLabels := map[Product]map[string]string{
		PolicyController:     nsLabels,
		KCC:                  nsLabels,
		ResourceGroup:        nsLabels,
		ConfigSyncMonitoring: nil,
		ConfigSync:           nil,
	}
	for product, ls := range productAndLabels {
		sources, err = b.logSourcesForProduct(ctx, product, listOps, ls)
		if err != nil {
			b.ErrorList = append(b.ErrorList, err)
		} else {
			toBeLogged = append(toBeLogged, sources...)
		}
	}

	// If we don't have any logs to pull down, report errors and exit
	if len(toBeLogged) == 0 {
		return nil
	}

	// Convert logSources to Readables
	toBeRead, errs := toBeLogged.convertLogSourcesToReadables(ctx, b.clientSet)
	b.ErrorList = append(b.ErrorList, errs...)

	return toBeRead
}

func (b *BugReporter) logSourcesForProduct(ctx context.Context, product Product, listOps client.ListOptions, nsLabels map[string]string) (logSources, error) {
	enabled := b.EnabledServices()

	ls, err := b.logSourcesForNamespace(ctx, productNamespaces[product], listOps, nsLabels)
	if err != nil {
		switch {
		case errorIs(err, missingNamespace) && !enabled[product]:
			glog.Infof("%s is not enabled", string(product))
			return nil, nil
		case errorIs(err, notManagedByACM) && !enabled[product]:
			glog.Infof("%s is not managed by ACM", string(product))
			return nil, nil
		case errorIs(err, notManagedByACM) && enabled[product]:
			glog.Errorf("%s is not managed by ACM, but it should be", string(product))
			return nil, err
		default:
			return nil, err
		}
	}
	if !enabled[product] {
		if len(ls) == 0 {
			glog.Infof("%s is not enabled", string(product))
		} else {
			glog.Infof("%s is not enabled but log sources found. It may be in the process of uninstalling. Adding logs to report.", string(product))
		}
	}
	return ls, err
}

func (b *BugReporter) logSourcesForNamespace(ctx context.Context, name string, listOps client.ListOptions, nsLabels map[string]string) (logSources, error) {
	fmt.Println("Retrieving " + name + " logs")
	ns, err := b.fetchNamespace(ctx, name, nsLabels)
	if err != nil {
		return nil, wrap(err, "failed to retrieve namespace %v", name)
	}

	pods, err := b.listPods(ctx, *ns, listOps)
	if err != nil {
		return nil, wrap(err, "failed to list pods for namespace %v", name)
	}

	return assembleLogSources(*ns, *pods), nil
}

func (b *BugReporter) fetchNamespace(ctx context.Context, name string, nsLabels map[string]string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	err := b.client.Get(ctx, types.NamespacedName{Name: name}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, newMissingNamespaceError(name)
		}
		return nil, fmt.Errorf("failed to get namespace with name=%v", name)
	}
	for k, v := range nsLabels {
		val, ok := ns.GetLabels()[k]
		if !ok || val != v {
			return nil, newNotManagedNamespaceError(name)
		}
	}
	return ns, nil
}

func (b *BugReporter) listPods(ctx context.Context, ns corev1.Namespace, lOps client.ListOptions) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	lOps.Namespace = ns.Name
	err := b.client.List(ctx, pods, &lOps)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pods for namespace %v", ns.Name)
	}

	return pods, nil
}

func assembleLogSources(ns corev1.Namespace, pods corev1.PodList) logSources {
	var ls logSources
	for _, p := range pods.Items {
		for _, c := range p.Spec.Containers {
			ls = append(ls, &logSource{
				ns:   ns,
				pod:  p,
				cont: c,
			})
		}
	}

	return ls
}

// resourcesToReadables is a type of function that converts the resources to readables.
type resourcesToReadables func(*unstructured.UnstructuredList, string) []Readable

// fetchResources provides a set of Readables for resources with a given group and version
// toReadables: the function that converts the resources to readables.
func (b *BugReporter) fetchResources(ctx context.Context, gv schema.GroupVersion, toReadables resourcesToReadables) (rd []Readable) {
	rl, err := b.clientSet.ServerResourcesForGroupVersion(gv.String())
	if err != nil {
		if errors.IsNotFound(err) {
			fmt.Printf("No %s resources found on cluster\n", gv.Group)
			return rd
		}
		if meta.IsNoMatchError(err) {
			fmt.Println("No match for " + gv.String())
			return rd
		}
		b.ErrorList = append(b.ErrorList, fmt.Errorf("failed to list server %s resources: %v", gv.Group, err))
		return rd
	}
	for _, apiResource := range rl.APIResources {
		// Check for empty singular name to skip subresources
		if apiResource.SingularName != "" {
			u := &unstructured.UnstructuredList{}
			u.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   gv.Group,
				Kind:    apiResource.SingularName,
				Version: gv.Version,
			})
			if err := b.client.List(ctx, u); err != nil {
				b.ErrorList = append(b.ErrorList, fmt.Errorf("failed to list %s resources: %v", apiResource.SingularName, err))
			} else {
				rd = append(rd, toReadables(u, apiResource.Name)...)
			}
		}
	}
	return rd
}

// FetchResources provides a set of Readables for configsync, configmanagement and resourcegroup resources.
func (b *BugReporter) FetchResources(ctx context.Context) []Readable {
	var rd []Readable
	var clusterResourceToReadables = func(u *unstructured.UnstructuredList, resourceName string) (r []Readable) {
		r = b.appendPrettyJSON(r, pathToClusterCmList(resourceName), u)
		return r
	}

	var namespacedResourceToReadables = func(u *unstructured.UnstructuredList, _ string) (r []Readable) {
		for _, o := range u.Items {
			r = b.appendPrettyJSON(r, pathToNamespacedResource(o.GetNamespace(), o.GetName()), o)
		}
		return r
	}

	// fetch cluster-scoped configmanagement resources
	cmReadables := b.fetchResources(ctx, v1.SchemeGroupVersion, clusterResourceToReadables)
	rd = append(rd, cmReadables...)

	namespaceGVs := []schema.GroupVersion{
		v1alpha1.SchemeGroupVersion,          // namespace-scoped configsync resources
		live.ResourceGroupGVK.GroupVersion(), // namespace-scoped resourcegroup resources
	}
	for _, gv := range namespaceGVs {
		readables := b.fetchResources(ctx, gv, namespacedResourceToReadables)
		rd = append(rd, readables...)
	}
	return rd
}

func pathToNamespacedResource(namespace, name string) string {
	return path.Join(Namespace, namespace, name)
}

func pathToClusterCmList(name string) string {
	return path.Join(ClusterScope, "configmanagement", name)
}

// FetchCMSystemPods provides a Readable for pods in the config-management-system,
// kube-system, resource-group-system and config-management-monitoring namespaces.
func (b *BugReporter) FetchCMSystemPods(ctx context.Context) (rd []Readable) {
	var namespaces = []string{
		configmanagement.ControllerNamespace,
		metav1.NamespaceSystem,
		RGControllerNamespace,
		metrics.MonitoringNamespace,
	}

	for _, ns := range namespaces {
		podList, err := b.clientSet.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			b.ErrorList = append(b.ErrorList, fmt.Errorf("failed to list %s pods: %v", ns, err))
		} else {
			rd = b.appendPrettyJSON(rd, pathToNamespacePodList(ns), podList)
		}
	}

	return rd
}

func pathToNamespacePodList(ns string) string {
	return path.Join(Namespace, ns, "pods")
}

// AddNomosStatusToZip writes `nomos status` to bugreport zip file
func (b *BugReporter) AddNomosStatusToZip(ctx context.Context) {
	if statusRc, err := status.GetStatusReadCloser(ctx, []string{b.k8sContext}); err != nil {
		b.ErrorList = append(b.ErrorList, err)
	} else if err = b.writeReadableToZip(Readable{
		Name:       path.Join(Processed, b.k8sContext, "status"),
		ReadCloser: statusRc,
	}); err != nil {
		b.WritingErrors = append(b.WritingErrors, err)
	}
}

// AddNomosVersionToZip writes `nomos version` to bugreport zip file
func (b *BugReporter) AddNomosVersionToZip(ctx context.Context) {
	if versionRc, err := version.GetVersionReadCloser(ctx, []string{b.k8sContext}); err != nil {
		b.ErrorList = append(b.ErrorList, err)
	} else if err = b.writeReadableToZip(Readable{
		Name:       path.Join(Processed, b.k8sContext, "version"),
		ReadCloser: versionRc,
	}); err != nil {
		b.WritingErrors = append(b.WritingErrors, err)
	}
}

func getReportName() string {
	now := time.Now()
	baseName := fmt.Sprintf("bug_report_%v.zip", now.Unix())
	nameWithPath, err := filepath.Abs(baseName)
	if err != nil {
		nameWithPath = baseName
	}

	return nameWithPath
}

func (b *BugReporter) writeReadableToZip(readable Readable) error {
	baseName := filepath.Base(b.name)
	dirName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	fileName := filepath.FromSlash(filepath.Join(dirName, readable.Name) + ".txt")
	f, err := b.writer.Create(fileName)
	if err != nil {
		e := fmt.Errorf("failed to create file %v inside zip: %v", fileName, err)
		return e
	}

	w := bufio.NewWriter(f)
	_, err = w.ReadFrom(readable.ReadCloser)
	if err != nil {
		e := fmt.Errorf("failed to write file %v to zip: %v", fileName, err)
		return e
	}

	err = w.Flush()
	if err != nil {
		e := fmt.Errorf("failed to flush writer to zip for file %v:i %v", fileName, err)
		return e
	}

	fmt.Println("Wrote file " + fileName)

	return nil
}

// WriteRawInZip writes raw kubernetes resource to bugreport zip file
func (b *BugReporter) WriteRawInZip(toBeRead []Readable) {

	for _, readable := range toBeRead {
		readable.Name = path.Join(Raw, b.k8sContext, readable.Name)
		err := b.writeReadableToZip(readable)
		if err != nil {
			b.WritingErrors = append(b.WritingErrors, err)
		}
	}

}

// Close closes all file streams
func (b *BugReporter) Close() {

	err := b.writer.Close()
	if err != nil {
		e := fmt.Errorf("failed to close zip writer: %v", err)
		b.ErrorList = append(b.ErrorList, e)
	}

	err = b.outFile.Close()
	if err != nil {
		e := fmt.Errorf("failed to close zip file: %v", err)
		b.ErrorList = append(b.ErrorList, e)
	}

	if len(b.WritingErrors) == 0 {
		glog.Infof("Bug report written to zip file: %v\n", b.name)
	} else {
		glog.Warningf("Some errors returned while writing zip file.  May exist at: %v\n", b.name)
	}
	b.ErrorList = append(b.ErrorList, b.WritingErrors...)

	if len(b.ErrorList) > 0 {
		for _, e := range b.ErrorList {
			glog.Errorf("Error: %v\n", e)
		}

		glog.Errorf("Partial bug report may have succeeded.  Look for file: %s\n", b.name)
	} else {
		fmt.Println("Created file " + b.name)
	}
}

// Open initializes bugreport zip files
func (b *BugReporter) Open() (err error) {
	b.name = getReportName()
	if b.outFile, err = os.Create(b.name); err != nil {
		return fmt.Errorf("failed to create file %v: %v", b.name, err)
	}
	b.writer = zip.NewWriter(b.outFile)
	return nil
}

func (b *BugReporter) appendPrettyJSON(rd []Readable, pathName string, object interface{}) []Readable {
	if data, err := json.MarshalIndent(object, "", "  "); err != nil {
		b.ErrorList = append(b.ErrorList, fmt.Errorf("invalid json response from resources %s: %v", pathName, err))
	} else {
		rd = append(rd, Readable{
			ReadCloser: ioutil.NopCloser(bytes.NewReader(data)),
			Name:       pathName,
		})
	}
	return rd
}
