package controllers

import (
	"io/ioutil"

	"sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
)

// deploymentConfig is defined in configmap manifests/templates/reconciler-manager/manifest.yaml.
var deploymentConfig = "deployment.yaml"

// nsParseDeployment parse deployment from deployment.yaml to deploy reconciler pod
// for root repository.
// Alias to enable test mocking.
var nsParseDeployment = func(de *appsv1.Deployment) error {
	return parseDeploymentFromConfig(deploymentConfig, de)
}

// rsParseDeployment parse deployment from deployment.yaml to deploy reconciler pod
// for root repository.
// Alias to enable test mocking.
var rsParseDeployment = func(de *appsv1.Deployment) error {
	return parseDeploymentFromConfig(deploymentConfig, de)
}

func parseDeploymentFromConfig(config string, de *appsv1.Deployment) error {
	// config is defined in manifests/templates/reconciler-manager/manifest.yaml
	yamlDep, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(yamlDep, de)
}
