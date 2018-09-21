/*
Copyright 2018 The Nomos Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/google/nomos/pkg/process/bash"
	"github.com/pkg/errors"
)

const (
	// scriptsDirectory is the directory relative to the working directory
	// that contains the utility shell scripts.  All scripts are expected to
	// be found in that directory.
	scriptsDirectory = "scripts"

	// certsDirectory is the directory relative to the working directory that
	// contains the generated certificates.
	certsDirectory = "certs"

	// yamlDirectory is the directory relative to workDir that contains the
	// YAML files to apply on a cluster.
	yamlDirectory = "manifests/deployment"
)

// certInstaller generates certificates and deploys secrets for admission
// controllers.
type certInstaller struct {
	// certScripts are scripts for certificate generation.  Should be replaced by
	// go-native generation scripts.
	generateScript string

	// deployScript is an admissionControllerScript used to deploy admission
	// controller specific secrets.
	deployScript string

	// subDir is the directory in the certs folder where controller
	// specific certificates are stored.
	subDir string
}

// deploySecrets deploys the generated certificates as secrets.
func (c *certInstaller) deploySecrets(workDir string) error {
	glog.V(5).Infof("deploySecrets: %s enter", c.name())

	certsPath := filepath.Join(workDir, certsDirectory, c.subDir)
	scriptPath := filepath.Join(workDir, scriptsDirectory, c.deployScript)
	yamlPath := filepath.Join(workDir, yamlDirectory)
	env := []string{
		fmt.Sprintf("SERVER_CERT_FILE=%v/server.crt", certsPath),
		fmt.Sprintf("SERVER_KEY_FILE=%v/server.key", certsPath),
		fmt.Sprintf("CA_CERT_FILE=%v/ca.crt", certsPath),
		fmt.Sprintf("CA_KEY_FILE=%v/ca.key", certsPath),
		fmt.Sprintf("YAML_DIR=%v", yamlPath),
	}
	if err := bash.RunWithEnv(context.Background(), scriptPath, env...); err != nil {
		return errors.Wrapf(err, "while admission controller secrets")
	}
	return nil
}

// createCertificates creates the certificates needed to bootstrap the syncing
// process.
func (c *certInstaller) createCertificates(workDir string) error {
	glog.V(5).Info("createCertificates: creating certificates")

	certsPath := filepath.Join(workDir, certsDirectory, c.subDir)
	glog.Infof("Creating certificates in path: %v", certsPath)
	err := os.MkdirAll(certsPath, os.ModeDir|0700)
	if err != nil {
		return errors.Wrapf(err, "while creating certs directory: %q", certsPath)
	}

	certgenScript := filepath.Join(workDir, scriptsDirectory, c.generateScript)
	if err := bash.RunWithEnv(context.Background(), certgenScript, fmt.Sprintf("OUTPUT_DIR=%v", certsPath)); err != nil {
		return errors.Wrapf(err, "while generating certificates")
	}
	return nil
}

// Returns the name of the certInstaller, which is also the subdirectory it
// stores the certificates in.
func (c *certInstaller) name() string {
	return c.subDir
}
