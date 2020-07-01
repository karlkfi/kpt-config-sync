package controllers

import (
	"io/ioutil"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/yaml"
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
