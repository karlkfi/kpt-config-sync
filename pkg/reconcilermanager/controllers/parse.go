package controllers

import (
	"io/ioutil"

	"sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
)

var (
	// deploymentConfig is defined in configmap manifests/templates/reconciler-manager-configmap.yaml
	deploymentConfig = "deployment.yaml"
)

// parseDeployment parse deployment from deployment.yaml to deploy reconciler pod
// Alias to enable test mocking.
var parseDeployment = func(de *appsv1.Deployment) error {
	return parseFromDeploymentConfig(deploymentConfig, de)
}

func parseFromDeploymentConfig(config string, obj *appsv1.Deployment) error {
	yamlDep, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(yamlDep, obj)
}
