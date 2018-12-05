/*
Copyright 2018 Google LLC.

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

package terraform

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

const (
	// The path to Terraform binary.
	binaryPath = "/bespin/terraform-bundle/terraform"

	// The path to Terraform plugin directory.
	pluginDir = "/bespin/terraform-bundle"

	defaultTmpDirPrefix = "terraform"

	// Default file name to use when generating Terraform config
	// file for bespin resources.
	defaultConfigFileName = "bespin_config.tf"

	// Default Terraform state file name.
	defaultStateFileName = "terraform.tfstate"

	// Default file mode when bespin creates a new file.
	defaultFilePerm = 0644

	// Terraform provider config.
	providerConfig = `provider "google" {
version = "1.19.1"
}
`
)

// Set to true if the bespin binary is running locally, otherwise
// assume it's running inside a container.
var local = flag.Bool("local", false, "True if running bespin controller locally")

func execCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, errors.Wrapf(err, "failed to execute command (%s): %s", name+" "+strings.Join(args[:], " "), stderr.String())
	}
	return out, nil
}

// Resource defines a collection of common Terraform-related functionalities
// that all bespin resources should implement.
type Resource interface {
	// GetTFResourceConfig converts the Project's Spec struct into Terraform config string.
	GetTFResourceConfig() (string, error)

	// GetTFImportConfig returns an empty Terraform resource block used for terraform import.
	GetTFImportConfig() string

	// GetTFResourceAddr returns the address of this resource in Terraform config.
	GetTFResourceAddr() string

	// GetID returns the resource ID from underlying provider (e.g. GCP).
	GetID() string
}

// Executor is a Terraform wrapper to run Terraform comamnds.
type Executor struct {
	// The working dir for this executor to perform all Terraform operations.
	dir string

	// The generated Terraform config file.
	configFileName string

	// Terraform local state file.
	stateFileName string

	// The bespin resource this executor runs against.
	resource Resource

	// The path to Terraform binary and plugin.
	binaryPath, pluginDir string

	// A map that stores the parsed output of "terraform state show".
	state map[string]string
}

// NewExecutor creates and approproately initializes a new Terraform executor.
func NewExecutor(resource Resource) (*Executor, error) {
	tmpDir, err := ioutil.TempDir("", defaultTmpDirPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tmp dir")
	}
	tfeBinaryPath, tfePluginDir := binaryPath, pluginDir
	if *local {
		tfeBinaryPath, tfePluginDir = "terraform", ""
	}
	tfe := &Executor{
		dir:            tmpDir,
		configFileName: defaultConfigFileName,
		stateFileName:  defaultStateFileName,
		resource:       resource,
		binaryPath:     tfeBinaryPath,
		pluginDir:      tfePluginDir,
	}
	glog.V(1).Infof("Terraform to use:\n - binary: %v\n - plugin-dir: %v", tfe.binaryPath, tfe.pluginDir)
	return tfe, nil
}

// Close removes the tmp working dir of the executor.
func (tfe *Executor) Close() error {
	glog.V(1).Infof("Removing terraform tmp dir %s", tfe.dir)
	if err := os.RemoveAll(tfe.dir); err != nil {
		return errors.Wrapf(err, "failed to remove tmp dir %s", tfe.dir)
	}
	return nil
}

// RunInit runs terraform init.
func (tfe *Executor) RunInit() error {
	glog.V(1).Infof("[%s]: Running terraform init.", tfe.dir)
	resourceConfig, err := tfe.resource.GetTFResourceConfig()
	if err != nil {
		return errors.Wrapf(err, "failed to get Terraform resource config from resource (%+v)", tfe.resource)
	}
	fileName := filepath.Join(tfe.dir, tfe.configFileName)
	err = ioutil.WriteFile(fileName, []byte(providerConfig+resourceConfig), defaultFilePerm)
	if err != nil {
		return errors.Wrapf(err, "failed to write Terraform resource config to file %s", fileName)
	}
	var out []byte
	args := []string{"init", "-input=false", "-upgrade=false"}
	if !*local {
		args = append(args, fmt.Sprintf("-plugin-dir=%s", pluginDir))
	}
	args = append(args, tfe.dir)
	out, err = execCommand(tfe.binaryPath, args...)
	if err != nil {
		return errors.Wrap(err, "failed to run terraform init")
	}
	glog.V(1).Infof("[%s]: Completed terraform init.\n%s", tfe.dir, out)
	return nil
}

// RunPlan runs terraform plan.
func (tfe *Executor) RunPlan() error {
	glog.V(1).Infof("[%s]: Running terraform plan.", tfe.dir)
	out, err := execCommand(
		tfe.binaryPath,
		"plan",
		fmt.Sprintf("-state=%s", filepath.Join(tfe.dir, tfe.stateFileName)),
		tfe.dir)
	if err != nil {
		return errors.Wrap(err, "failed to run terraform plan")
	}
	glog.V(1).Infof("[%s]: Completed terraform plan.\n%s", tfe.dir, out)
	return nil
}

// RunPlanDestroy runs terraform plan.
func (tfe *Executor) RunPlanDestroy() error {
	glog.V(1).Infof("[%s]: Running terraform plan destroy.", tfe.dir)
	out, err := execCommand(
		tfe.binaryPath,
		"plan",
		"-destroy",
		fmt.Sprintf("-state=%s", filepath.Join(tfe.dir, tfe.stateFileName)),
		tfe.dir)
	if err != nil {
		return errors.Wrap(err, "failed to run terraform plan -destroy")
	}
	glog.V(1).Infof("Completed terraform plan -destroy.\n%s", out)
	return nil
}

// RunApply runs terraform apply.
func (tfe *Executor) RunApply() error {
	glog.V(1).Infof("[%s]: Running terraform apply.", tfe.dir)
	out, err := execCommand(
		tfe.binaryPath,
		"apply",
		"-auto-approve",
		"-refresh=true",
		fmt.Sprintf("-state=%s", filepath.Join(tfe.dir, tfe.stateFileName)),
		fmt.Sprintf("-state-out=%s", filepath.Join(tfe.dir, tfe.stateFileName)),
		tfe.dir)
	if err != nil {
		return errors.Wrap(err, "failed to run terraform apply")
	}
	glog.V(1).Infof("[%s]: Completed terraform apply.\n%s", tfe.dir, out)
	return nil
}

// RunImport imports the attached resource into local Terraform state.
func (tfe *Executor) RunImport() error {
	glog.V(1).Infof("[%s]: Running terraform import.", tfe.dir)
	fileName := filepath.Join(tfe.dir, tfe.configFileName)
	err := ioutil.WriteFile(fileName, []byte(tfe.resource.GetTFImportConfig()), defaultFilePerm)
	if err != nil {
		return errors.Wrapf(err, "failed to write terraform import config to file %s", fileName)
	}
	out, err := execCommand(
		tfe.binaryPath,
		"import",
		fmt.Sprintf("-config=%s", tfe.dir), // Dir to find provider info.
		fmt.Sprintf("-state=%s", filepath.Join(tfe.dir, tfe.stateFileName)),     // Source state file.
		fmt.Sprintf("-state-out=%s", filepath.Join(tfe.dir, tfe.stateFileName)), // Target state file to update.
		tfe.resource.GetTFResourceAddr(),
		tfe.resource.GetID())
	if err != nil {
		glog.Warningf("failed to run terraform import: %v", err)
	}
	glog.V(1).Infof("[%s]: Completed terraform import.\n%s", tfe.dir, out)
	return nil
}

func run(f func() error, err error) error {
	if err != nil {
		return err
	}
	return f()
}

// RunDestroy runs terraform destroy to remove the resource.
func (tfe *Executor) RunDestroy() error {
	glog.V(1).Infof("[%s]: Running terraform destroy.", tfe.dir)
	out, err := execCommand(
		tfe.binaryPath,
		"destroy",
		"-auto-approve",
		fmt.Sprintf("-state=%s", filepath.Join(tfe.dir, tfe.stateFileName)),     // Source state file.
		fmt.Sprintf("-state-out=%s", filepath.Join(tfe.dir, tfe.stateFileName)), // Target state file to update.
		tfe.dir)
	if err != nil {
		return errors.Wrap(err, "failed to run terraform destroy")
	}
	glog.V(1).Infof("Completed terraform destroy.\n%s", out)
	return nil
}

// RunCreateOrUpdateFlow runs most common Terraform commands sequence in the order of init/import/plan/apply
// to create or update resource.
// TODO(b/120279113): Ideally we should read the error from RunImport stdout/stderr and do something smarter here,
// but letting the operation fail further down is fine for now.
func (tfe *Executor) RunCreateOrUpdateFlow() error {
	var err error
	err = run(tfe.RunInit, err)
	err = run(tfe.RunImport, err)
	err = run(tfe.RunInit, err)
	err = run(tfe.RunPlan, err)
	err = run(tfe.RunApply, err)
	return err
}

// UpdateState inspects the terraform local state file and updates the resource info
// in the executor's map, e.g. {"id": "xxx", "create_time": "xxx", ...}. The caller of this function
// needs to make sure the terraform state file is update-to-date otherwise the stale data
// maybe used later.
func (tfe *Executor) UpdateState() error {
	glog.V(1).Infof("[%s]: Running terraform state show.", tfe.dir)
	resourceAddr := tfe.resource.GetTFResourceAddr()
	out, err := execCommand(
		tfe.binaryPath,
		"state",
		"show",
		fmt.Sprintf("-state=%s", filepath.Join(tfe.dir, tfe.stateFileName)),
		resourceAddr)
	if err != nil {
		return errors.Wrapf(err, "failed to run terraform state show on resource %s", resourceAddr)
	}
	glog.V(1).Infof("Done terraform state show on resource %s.\n%s", resourceAddr, out)

	m, err := parseStateConfig(string(out))
	if err != nil {
		return errors.Wrap(err, "failed to parse terraform state")
	}
	tfe.state = m
	return nil
}

// parseStateConfig parses the input string with expected format and
// returns a map of key->value pairs.
// config is of format (using Folder as an example):
// id              = folders/1234567
// create_time     = 2000-12-05T21:32:29.614Z
// display_name    = name-of-the-resource
// lifecycle_state = ACTIVE
// name            = folders/1234567
// parent          = organizations/9876543
func parseStateConfig(config string) (map[string]string, error) {
	m := make(map[string]string)
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.Trim(line, " ")
		if len(line) == 0 {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			glog.Errorf("config line doesn't match known format: [key] = [value]: %s", line)
			return nil, fmt.Errorf("config line doesn't match known format: [key] = [value]: %s", line)
		}
		m[strings.Trim(kv[0], " ")] = strings.Trim(kv[1], " ")
	}
	return m, nil
}

// GetFolderID returns the Folder ID stored in its state map.
func (tfe *Executor) GetFolderID() (int, error) {
	re := regexp.MustCompile(`^folders\/(\d+)$`)
	if fid, ok := tfe.state["id"]; ok {
		// "id": "folders/1234567"
		switch {
		case re.MatchString(fid):
			id, err := strconv.Atoi(strings.Split(fid, "/")[1])
			if err != nil {
				return 0, errors.Wrapf(err, "failed to get Folder ID: %s", fid)
			}
			return id, nil
		default:
			return 0, fmt.Errorf("invalid Folder ID: %s", fid)
		}
	}
	return 0, fmt.Errorf("Folder ID not found")
}

// RunDeleteFlow runs Terraform commands sequence in the order of init/import/destroy to delete resource.
// TODO(b/120279113): Ideally we should read the error from RunImport stdout/stderr and do something smarter here,
// but letting the operation fail further down is fine for now.
func (tfe *Executor) RunDeleteFlow() error {
	var err error
	err = run(tfe.RunInit, err)
	err = run(tfe.RunImport, err)
	err = run(tfe.RunInit, err)
	err = run(tfe.RunPlanDestroy, err)
	err = run(tfe.RunDestroy, err)
	return err
}
