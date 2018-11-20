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

package util

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// bespinLocalRun is true meaning the bespin binary is running locally, otherwise
// we assume it's running inside a container.
var bespinLocalRun = flag.Bool("local_run", false, "True if running bespin controller locally")

// RunTerraform takes a terraform-formated string tfs and does the following:
// 1. create tmp dir as terraform's working dir and change current dir to it
// 2. write tfs to a tmp file.
// 3. run terrafirm init, plan, apply
func RunTerraform(tfs string) error {
	terraformBinary := "/bespin/terraform-bundle/terraform"
	terraformPluginDir := "/bespin/terraform-bundle"
	if *bespinLocalRun {
		terraformBinary = "terraform"
		terraformPluginDir = ""
	}
	glog.V(1).Infof("Terraform to use:\n - binary: %v\n - plugin-dir: %v", terraformBinary, terraformPluginDir)

	dir, err := ioutil.TempDir("/tmp", "terraform")
	if err != nil {
		return errors.Wrap(err, "error in creating tmp dir")
	}
	defer func() {
		err = os.RemoveAll(dir)
		if err != nil {
			glog.Errorf("failed to remove temp dir after running terraform\n")
			err = errors.Wrap(err, "failed to remove temp dir after running terraform")
		}
	}()

	tmptf := filepath.Join(dir, "gcp_crd.tf")
	f, err := os.Create(tmptf)
	glog.V(1).Infof("Created tmp tf file %v", tmptf)
	if err != nil {
		return errors.Wrap(err, "error in creating tmp tf file")
	}

	_, err = fmt.Fprintf(f, tfs)
	if err != nil {
		return errors.Wrap(err, "error writing tf to file")
	}
	err = f.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close terraform file")
	}
	var stderr bytes.Buffer
	glog.V(1).Infof("Running terraform init in dir %s\n", dir)
	cmd := exec.Command(terraformBinary, "init", "-plugin-dir", terraformPluginDir, dir)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		glog.Errorf("terraform init failed with err: %s\n", stderr.String())
		return errors.Wrap(err, "terraform init error")
	}
	glog.V(1).Infof("Done terraform init %s\n", out)

	glog.V(1).Infof("Running terraform plan in dir %s\n", dir)
	cmd = exec.Command(terraformBinary, "plan", dir)
	cmd.Stderr = &stderr
	out, err = cmd.Output()
	if err != nil {
		glog.Errorf("terraform plan failed with err: %s\n", stderr.String())
		return errors.Wrap(err, "terraform plan error")
	}
	glog.V(1).Infof("Done terraform plan %s\n", out)

	glog.V(1).Infof("Running terraform apply in dir %s\n", dir)
	cmd = exec.Command(terraformBinary, "apply", "-auto-approve", dir)
	cmd.Stderr = &stderr
	out, err = cmd.Output()
	if err != nil {
		glog.Errorf("terraform apply failed with err: %s\n", stderr.String())
		return errors.Wrap(err, "terraform apply error")
	}
	glog.V(1).Infof("Done terraform apply %s\n", out)

	return err
}
