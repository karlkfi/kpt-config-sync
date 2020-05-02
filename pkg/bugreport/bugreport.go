package bugreport

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/nomos/cmd/nomos/status"
	"github.com/google/nomos/cmd/nomos/util"
	"github.com/google/nomos/cmd/nomos/version"
	"github.com/google/nomos/pkg/api/configmanagement"
	"github.com/google/nomos/pkg/client/restconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// BugReporter handles basic data gathering tasks for generating a
// bug report
type BugReporter struct {
	client    client.Reader
	clientSet *kubernetes.Clientset
	ctx       context.Context
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
		Group: util.ConfigManagementGroup,
		// we rely on a specific version implicitly later in the code, so this should
		// be hardcoded
		Version: "v1",
		Kind:    util.ConfigManagementKind,
	})

	errorList := []error{}
	currentk8sContext, err := restconfig.CurrentContextName()
	if err != nil {
		errorList = append(errorList, err)
	}

	if err := c.Get(ctx, types.NamespacedName{Name: util.ConfigManagementName}, cm); err != nil {
		return nil, err
	}
	zipName := getReportName()
	outFile, err := os.Create(zipName)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %v: %v", zipName, err)
	}
	return &BugReporter{
		client:        c,
		clientSet:     cs,
		ctx:           ctx,
		cm:            cm,
		k8sContext:    currentk8sContext,
		outFile:       outFile,
		writer:        zip.NewWriter(outFile),
		name:          zipName,
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
		// Same for Config Sync, though here the "disabled" condition is if the git
		// config is "empty", which involves looking for an empty proxy config
		syncGitCfg, _, _ := unstructured.NestedMap(b.cm.Object, "spec", "git")
		configSyncEnabled := false
		for k := range syncGitCfg {
			if k != "proxy" {
				configSyncEnabled = true
			}
		}
		proxy, _, _ := unstructured.NestedMap(syncGitCfg, "proxy")
		if len(proxy) > 0 {
			configSyncEnabled = true
		}
		enabled[ConfigSync] = configSyncEnabled
		b.enabled = enabled
	}

	return b.enabled
}

// FetchLogSources provides a set of Readables for all of nomos' container logs
// TODO: Still need to figure out a good way to test this
func (b *BugReporter) FetchLogSources() ([]Readable, []error) {
	var toBeLogged logSources
	var errorList []error

	// for each namespace, generate a list of logSources
	listOps := client.ListOptions{LabelSelector: labels.NewSelector().Add(operatorLabelSelectorOrDie())}
	sources, err := b.logSourcesForNamespace("kube-system", listOps, nil)
	if err != nil {
		errorList = append(errorList, err)
	} else {
		toBeLogged = append(toBeLogged, sources...)
	}

	listOps = client.ListOptions{}
	nsLabels := map[string]string{"configmanagement.gke.io/configmanagement": "config-management"}
	sources, err = b.logSourcesForProduct(PolicyController, listOps, nsLabels)
	if err != nil {
		errorList = append(errorList, err)
	} else {
		toBeLogged = append(toBeLogged, sources...)
	}

	sources, err = b.logSourcesForProduct(KCC, listOps, nsLabels)
	if err != nil {
		errorList = append(errorList, err)
	} else {
		toBeLogged = append(toBeLogged, sources...)
	}

	sources, err = b.logSourcesForProduct(ConfigSync, listOps, nil)
	if err != nil {
		errorList = append(errorList, err)
	} else {
		toBeLogged = append(toBeLogged, sources...)
	}

	// If we don't have any logs to pull down, report errors and exit
	if len(toBeLogged) == 0 {
		return nil, errorList
	}

	// Convert logSources to Readables
	toBeRead, errs := toBeLogged.convertLogSourcesToReadables(b.clientSet)
	errorList = append(errorList, errs...)

	return toBeRead, errorList
}

func (b *BugReporter) logSourcesForProduct(product Product, listOps client.ListOptions, nsLabels map[string]string) (logSources, error) {
	enabled := b.EnabledServices()

	ls, err := b.logSourcesForNamespace(productNamespaces[product], listOps, nsLabels)
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
		glog.Infof("%s is not enabled but log sources found. It may be in the process of uninstalling. Adding logs to report.", string(product))
	}
	return ls, err
}

func (b *BugReporter) logSourcesForNamespace(name string, listOps client.ListOptions, nsLabels map[string]string) (logSources, error) {
	fmt.Println("Retrieving " + name + " logs")
	ns, err := b.fetchNamespace(name, nsLabels)
	if err != nil {
		return nil, wrap(err, "failed to retrieve namespace %v", name)
	}

	pods, err := b.listPods(*ns, listOps)
	if err != nil {
		return nil, wrap(err, "failed to list pods for namespace %v", name)
	}

	return assembleLogSources(*ns, *pods), nil
}

func (b *BugReporter) fetchNamespace(name string, nsLabels map[string]string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	err := b.client.Get(b.ctx, types.NamespacedName{Name: name}, ns)
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

func (b *BugReporter) listPods(ns corev1.Namespace, lOps client.ListOptions) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	lOps.Namespace = ns.Name
	err := b.client.List(b.ctx, pods, &lOps)
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

// FetchCMResources provides a set of Readables for configmanagement resources
// TODO convert json content to yaml
func (b *BugReporter) FetchCMResources() (rd []Readable, errorList []error) {
	base := "apis/configmanagement.gke.io/v1"
	var resourceLists = []string{
		"clusterconfigs",
		"namespaceconfigs",
		"syncs",
		"configmanagements",
		"repos",
	}

	for _, listName := range resourceLists {
		if raw, err := b.clientSet.CoreV1().RESTClient().Get().AbsPath(path.Join(base, listName)).Stream(); err != nil {
			errorList = append(errorList, fmt.Errorf("failed to list %s resources: %v", listName, err))
		} else {
			rd = append(rd, Readable{
				ReadCloser: raw,
				Name:       path.Join("cluster-scope", "configmanagement", listName),
			})
		}

	}
	return rd, errorList
}

// FetchCMSystemPods provides a Readable for pods in the config management system and kube-system namespaces
func (b *BugReporter) FetchCMSystemPods() (rd []Readable, errorList []error) {
	var namespaces = []string{
		configmanagement.ControllerNamespace,
		"kube-system",
	}
	for _, ns := range namespaces {
		rc, err := b.clientSet.CoreV1().RESTClient().Get().AbsPath(path.Join("api/v1/namespaces", ns, "pods")).Stream()
		if err != nil {
			errorList = append(errorList, fmt.Errorf("failed to list %s pods: %v", ns, err))
		} else {
			rd = append(rd, Readable{
				ReadCloser: rc,
				Name:       path.Join(ns, "pods"),
			})
		}
	}

	return rd, errorList
}

// AddNomosStatusToZip writes `nomos status` to bugreport zip file
func (b *BugReporter) AddNomosStatusToZip() {
	if statusRc, err := status.GetStatusReadCloser([]string{b.k8sContext}); err != nil {
		b.ErrorList = append(b.ErrorList, err)
	} else if err = b.writeReadableToZip(Readable{
		Name:       path.Join("processed", b.k8sContext, "status"),
		ReadCloser: statusRc,
	}); err != nil {
		b.WritingErrors = append(b.WritingErrors, err)
	}
}

// AddNomosVersionToZip writes `nomos version` to bugreport zip file
func (b *BugReporter) AddNomosVersionToZip() {
	if versionRc, err := version.GetVersionReadCloser([]string{b.k8sContext}); err != nil {
		b.ErrorList = append(b.ErrorList, err)
	} else if err = b.writeReadableToZip(Readable{
		Name:       path.Join("processed", b.k8sContext, "version"),
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
func (b *BugReporter) WriteRawInZip(toBeRead []Readable, errs []error) {
	if len(errs) > 0 {
		b.ErrorList = append(b.ErrorList, errs...)
	}

	for _, readable := range toBeRead {
		readable.Name = path.Join("raw", b.k8sContext, readable.Name)
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
