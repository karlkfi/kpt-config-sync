package controllers

import (
	"io/ioutil"

	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	v1 "github.com/google/nomos/pkg/api/configmanagement/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func parseDeployment(de *appsv1.Deployment) error {
	// deployment.yaml is defined in manifests/templates/reconciler-manager/manifest.yaml
	yamlDep, err := ioutil.ReadFile("deployment.yaml")
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlDep, de)
	if err != nil {
		return err
	}
	return err
}

func configMapData(branch, repo string) map[string]string {
	result := make(map[string]string)
	result["GIT_KNOWN_HOSTS"] = "false"
	result["GIT_SYNC_BRANCH"] = branch
	result["GIT_SYNC_REPO"] = repo
	result["GIT_SYNC_REV"] = "HEAD"
	result["GIT_SYNC_WAIT"] = "15"
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
