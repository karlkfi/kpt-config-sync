package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"

	"github.com/google/nomos/pkg/api/configsync/v1beta1"
	"github.com/google/nomos/pkg/core"
	"sigs.k8s.io/yaml"

	appsv1 "k8s.io/api/apps/v1"
)

var (
	// deploymentConfig is defined in configmap manifests/templates/reconciler-manager-configmap.yaml
	deploymentConfig = "deployment.yaml"
	// deploymentConfigChecksumAnnotationKey tracks the checksum of the content in `deploymentConfig`.
	deploymentConfigChecksumAnnotationKey = v1beta1.ConfigSyncPrefix + "config-checksum"
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

	if err := yaml.Unmarshal(yamlDep, obj); err != nil {
		return err
	}

	sum := sha256.Sum256(yamlDep)
	sumString := hex.EncodeToString(sum[:])
	core.SetAnnotation(obj, deploymentConfigChecksumAnnotationKey, sumString)
	return nil
}
