package bugreport

import (
	"fmt"

	"github.com/google/nomos/pkg/api/configmanagement"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type coreClient interface {
	CoreV1() corev1.CoreV1Interface
}

// FetchLogSources provides a set of Readables for all of nomos' container logs
// TODO: Still need to figure out a good way to test this
func FetchLogSources(client coreClient) ([]Readable, []error) {
	var toBeLogged logSources
	var errorList []error

	// for each namespace, generate a list of logSources
	listOps := metav1.ListOptions{LabelSelector: "k8s-app=config-management-operator"}
	sources, err := logSourcesForNamespace(client, "kube-system", listOps)
	if err != nil {
		errorList = append(errorList, err)
	} else {
		toBeLogged = append(toBeLogged, sources...)
	}

	listOps = metav1.ListOptions{}
	sources, err = logSourcesForNamespace(client, configmanagement.ControllerNamespace, listOps)
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
	toBeRead, errs := toBeLogged.convertLogSourcesToReadables(client)
	errorList = append(errorList, errs...)

	return toBeRead, errorList
}

func logSourcesForNamespace(cs coreClient, name string, listOps metav1.ListOptions) (logSources, error) {
	ns, err := fetchNamespace(cs, name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve namespace %v: %v", name, err)
	}

	pods, err := listPods(cs, *ns, listOps)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for namespace %v: %v", name, err)
	}

	return assembleLogSources(*ns, *pods), nil
}

func assembleLogSources(ns v1.Namespace, pods v1.PodList) logSources {
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

func fetchNamespace(client coreClient, name string) (*v1.Namespace, error) {
	ns, err := client.CoreV1().Namespaces().Get(name, metav1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("failed to get namespace with name=%v", name)
	}

	return ns, nil
}

func listPods(cs coreClient, ns v1.Namespace, lOps metav1.ListOptions) (*v1.PodList, error) {
	pods, err := cs.CoreV1().Pods(ns.Name).List(lOps)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pods for namespace %v", ns.Name)
	}

	return pods, nil
}
