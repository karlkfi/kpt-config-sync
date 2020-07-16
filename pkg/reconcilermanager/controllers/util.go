package controllers

import (
	"io/ioutil"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO b/161890483 Initiliaze the deployment object to be used while
// initializing the reconciler struct in the manager.
func parseDeployment(config string, de *appsv1.Deployment) error {
	// config is defined in manifests/templates/reconciler-manager/manifest.yaml
	yamlDep, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlDep, de)
	if err != nil {
		return err
	}
	return err
}

func gitSyncData(branch, repo string) map[string]string {
	result := make(map[string]string)
	result["GIT_KNOWN_HOSTS"] = "false"
	result["GIT_SYNC_BRANCH"] = branch
	result["GIT_SYNC_REPO"] = repo
	result["GIT_SYNC_REV"] = "HEAD"
	result["GIT_SYNC_WAIT"] = "15"
	return result
}

func importerData(policydir string) map[string]string {
	result := make(map[string]string)
	result["POLICY_DIR"] = policydir
	return result
}

func sourceFormatData(format string) map[string]string {
	result := make(map[string]string)
	result["SOURCE_FORMAT"] = format
	return result
}

func ownerReference(kind, name string, uid types.UID) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		{
			APIVersion:         v1.SchemeGroupVersion.String(),
			Kind:               kind,
			Name:               name,
			Controller:         pointer.BoolPtr(true),
			BlockOwnerDeletion: pointer.BoolPtr(true),
			UID:                uid,
		},
	}
}

func envFromSources(configmapRef map[string]*bool) []corev1.EnvFromSource {
	var envFromSource []corev1.EnvFromSource
	for name, optional := range configmapRef {
		cfgMap := corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				Optional: optional,
			},
		}
		envFromSource = append(envFromSource, cfgMap)
	}
	return envFromSource
}

func buildRepoSyncName(names ...string) string {
	prefix := []string{repoSyncReconcilerPrefix}
	return strings.Join(append(prefix, names...), "-")
}
