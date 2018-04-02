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

// Package installer contains the business logic of the installer.
//
// TODO(fmil): The installer should be a self-sufficient go binary and not rely
// on kubectl and others.
package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/google/nomos/pkg/process/bash"
	"github.com/google/nomos/pkg/process/kubectl"
	"github.com/pkg/errors"
)

const (
	// yamlDirectory is the directory relative to workDir that contains the
	// YAML files to apply on a cluster.
	yamlDirectory = "yaml"

	// scriptsDirectory is the directory relative to the working directory
	// that contains the utility shell scripts.  All scripts are expected to
	// be found in that directory.
	scriptsDirectory = "scripts"

	// certsDirectory is the directory relative to the working directory that
	// contains the generated certificates.
	certsDirectory = "certs"

	// certScript is a script for certificate generation.  Should be replaced
	// by go-native generation script.
	certScript = "generate-resourcequota-admission-controller-certs.sh"

	// admissionControllerScript is a admissionControllerScript used to deploy
	// admission controller specific secrets.
	admissionControllerScript = "deploy-resourcequota-admission-controller.sh"

	// defaultNamespace is the namespace to place resources into.
	defaultNamespace = "nomos-system"

	// deploymentTimeout is the default timeout to wait for each Nomos
	// component to become functional.  Components initialize in parallel, so
	// we expect that it's unlikely a wait would be
	// deploymentTimeout*len(deploymentComponents).
	deploymentTimeout = 3 * time.Minute
)

var (
	// deploymentComponents are the components that are expected to be running
	// after installer completes.
	deploymentComponents = []string{
		fmt.Sprintf("%v:git-policy-importer", defaultNamespace),
		fmt.Sprintf("%v:resourcequota-admission-controller", defaultNamespace),
		fmt.Sprintf("%v:syncer", defaultNamespace),
	}

	// deploymentClusterRolesAndBindings are the cluster roles and
	// clusterrolebindings names created for the nomos system.
	deploymentClusterRolesAndBindings = []string{
		"nomos-nomosresourcequota-controller",
		"nomos-policy-importer",
		"nomos-resourcequota-admission-controller",
		"nomos-syncer",
	}

	// mv is the minimum supported cluster version.  It is not possible to install
	// on an earlier cluster due to missing features.
	mv = semver.MustParse("1.9.0")
)

// Installer is the process that runs the system installation.
type Installer struct {
	// The configuration that the installer works out of.
	c config.Config

	// The working directory of the installer.
	workDir string
}

// New returns a new Installer instance.
func New(c config.Config, workDir string) *Installer {
	return &Installer{c: c, workDir: workDir}
}

// createCertificates creates the certificates needed to bootstrap the syncing
// process.
func (i *Installer) createCertificates() error {
	glog.V(5).Info("createCertificates: creating certificates")
	certsPath := filepath.Join(i.workDir, certsDirectory)
	err := os.MkdirAll(certsPath, os.ModeDir|0700)
	if err != nil {
		return errors.Wrapf(err, "while creating certs directory: %q", certsPath)
	}
	certgenScript := filepath.Join(i.workDir, scriptsDirectory, certScript)
	if err := bash.RunWithEnv(context.Background(), certgenScript, fmt.Sprintf("OUTPUT_DIR=%v", certsPath)); err != nil {
		return errors.Wrapf(err, "while generating certificates")
	}
	return nil
}

// applyAll runs 'kubectl apply -f applyDir'.
func (i *Installer) applyAll(applyDir string) error {
	kc := kubectl.New(context.Background())
	glog.Infof("applying YAML files from directory: %v", applyDir)
	fi, err := os.Stat(applyDir)
	if err != nil {
		return errors.Wrapf(err, "applyAll: stat %v", applyDir)
	}
	if !fi.IsDir() {
		return errors.Errorf("applyAll: not a directory: %v", applyDir)
	}
	return kc.Apply(applyDir)
}

func (i *Installer) deploySshSecrets() error {
	const secret = "git-creds"
	glog.V(5).Info("deploySshSecrets: enter")
	if i.c.Ssh.PrivateKeyFilename == "" {
		glog.V(5).Infof("Not deploying SSH secrets, config has none.")
		return nil
	}
	c := kubectl.New(context.Background())
	c.DeleteSecret(secret, defaultNamespace)
	// TODO(filmil): Should there be more validation of these file paths?
	if err := c.CreateSecretGenericFromFile(
		secret, defaultNamespace,
		fmt.Sprintf("ssh=%v", i.c.Ssh.PrivateKeyFilename),
		fmt.Sprintf("known_hosts=%v", i.c.Ssh.KnownHostsFilename)); err != nil {
		return errors.Wrapf(err, "while creating ssh secrets")
	}
	return nil
}

func (i *Installer) deployGitConfig() error {
	const configmap = "git-policy-importer"
	glog.V(5).Info("deployGitConfig: enter")
	if i.c.Git.SyncRepo == "" {
		glog.V(5).Info("Not deploying git configuration, no config specified.")
		return nil
	}
	c := kubectl.New(context.Background())
	if err := c.DeleteConfigmap(configmap, defaultNamespace); err != nil {
		return errors.Wrapf(err, "while deleting configmap git-policy-importer")
	}

	if err := c.CreateConfigmapFromLiterals(
		configmap, defaultNamespace,
		"GIT_SYNC_SSH=true",
		fmt.Sprintf("GIT_SYNC_REPO=%v", i.c.Git.SyncRepo),
		fmt.Sprintf("GIT_SYNC_BRANCH=%v", i.c.Git.SyncBranch),
		fmt.Sprintf("GIT_SYNC_WAIT=%v", i.c.Git.SyncWaitSeconds),
		fmt.Sprintf("POLICY_DIR=%v", i.c.Git.RootPolicyDir),
	); err != nil {
		return errors.Wrapf(err, "while creating configmap git-policy-importer")
	}
	return nil
}

func (i *Installer) deployResourceQuotaSecrets() error {
	glog.V(5).Info("deployResourceQuotaSecrets: enter")

	certsPath := filepath.Join(i.workDir, certsDirectory)
	scriptPath := filepath.Join(i.workDir, scriptsDirectory, admissionControllerScript)
	yamlPath := filepath.Join(i.workDir, yamlDirectory)
	env := []string{
		fmt.Sprintf("SERVER_CERT_FILE=%v/server.crt", certsPath),
		fmt.Sprintf("SERVER_KEY_FILE=%v/server.key", certsPath),
		fmt.Sprintf("CA_CERT_FILE=%v/ca.crt", certsPath),
		fmt.Sprintf("CA_KEY_FILE=%v/ca.key", certsPath),
		fmt.Sprintf("YAML_DIR=%v", yamlPath),
	}
	if err := bash.RunWithEnv(context.Background(), scriptPath, env...); err != nil {
		return errors.Wrapf(err, "while creating admission controller secrets")
	}
	return nil
}

func (i *Installer) deploySecrets() error {
	glog.V(5).Info("deploySecrets: enter")
	err := i.deploySshSecrets()
	if err != nil {
		return errors.Wrapf(err, "while deploying ssh secrets")
	}
	err = i.deployResourceQuotaSecrets()
	if err != nil {
		return errors.Wrapf(err, "while deploying resource quota secrets")
	}
	return nil
}

func (i *Installer) addClusterAdmin(user string) error {
	glog.V(5).Infof("Adding %q as cluster admin", user)
	c := kubectl.New(context.Background())
	return c.AddClusterAdmin(user)
}

func (i *Installer) removeClusterAdmin(user string) error {
	glog.V(5).Infof("Removing %q as cluster admin", user)
	c := kubectl.New(context.Background())
	return c.RemoveClusterAdmin(user)
}

// checkVersion checks whether the cluster's kubernetes version is recent enough to
// support nomos.
func (i *Installer) checkVersion(ctx *kubectl.Context) error {
	v, err := ctx.GetClusterVersion()
	if err != nil {
		return errors.Wrapf(err, "could not check version")
	}
	if v.LT(mv) {
		return errors.Errorf("detected cluster version: %v is less than minimum: %v", v, mv)
	}
	return nil
}

// processCluster installs the necessary files on the currently active cluster.
// In addition the current cluster context is passed in.
func (i *Installer) processCluster(cluster string) error {
	var err error
	glog.V(5).Info("processCluster: enter")

	if i.c.User != "" {
		if err = i.addClusterAdmin(i.c.User); err != nil {
			return errors.Wrapf(err, "could not make %v the cluster admin.", i.c.User)
		}
		defer func() {
			// Ensure that this is ran at end of cluster process, irrespective
			// of whether the install was successful.
			if err := i.removeClusterAdmin(i.c.User); err != nil {
				glog.Warningf("could not remove cluster admin role for user: %v: %v", i.c.User, err)
			}
		}()
	}
	c := kubectl.New(context.Background())

	if err := i.checkVersion(c); err != nil {
		return errors.Wrapf(err, "while checking version for context")
	}
	// Delete the git policy importer deployment.  This is important because a
	// change in the git creds should also be reflected in the importer.
	if err = c.DeleteDeployment("git-policy-importer", defaultNamespace); err != nil {
		return errors.Wrapf(err, "while deleting git-policy-importer deployment")
	}
	// The common manifests need to be applied first, as they create the
	// namespace.
	if err = i.applyAll(filepath.Join(i.workDir, "manifests/common")); err != nil {
		return errors.Wrapf(err, "while applying manifests/common")
	}
	if err = i.deployGitConfig(); err != nil {
		return errors.Wrapf(err, "processCluster")
	}
	if err = i.deploySecrets(); err != nil {
		return errors.Wrapf(err, "processCluster")
	}
	if err = i.applyAll(filepath.Join(i.workDir, "manifests/enrolled")); err != nil {
		return errors.Wrapf(err, "while applying manifests/enrolled")
	}
	if err = i.applyAll(filepath.Join(i.workDir, "yaml")); err != nil {
		return errors.Wrapf(err, "while applying yaml")
	}
	if err = c.WaitForDeployments(deploymentTimeout, deploymentComponents...); err != nil {
		return errors.Wrapf(err, "while waiting for system components")
	}
	return err
}

// restoreContext sets the context to a (previously current) context.
func restoreContext(c string) error {
	k := kubectl.New(context.Background())
	if err := k.SetContext(c); err != nil {
		return errors.Wrapf(err, "while restoring context: %q", err)
	}
	return nil
}

// Run starts the installer process, and reports error at the process end, if any.
func (i *Installer) Run() error {
	if len(i.c.Contexts) == 0 {
		return errors.Errorf("no clusters requested for installation")
	}

	cl, err := kubectl.LocalClusters()
	defer func() {
		if err := restoreContext(cl.Current); err != nil {
			glog.Errorf("while restoring context: %q: %v", cl.Current, err)
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "while getting local list of clusters")
	}
	i.createCertificates()
	kc := kubectl.New(context.Background())
	for _, cluster := range i.c.Contexts {
		glog.Infof("processing cluster: %q", cluster)
		err := kc.SetContext(cluster)
		if err != nil {
			return errors.Wrapf(err, "while setting context: %q", cluster)
		}
		// The processed cluster is set through the context use.
		err = i.processCluster(cluster)
		if err != nil {
			return errors.Wrapf(err, "while processing cluster: %q", cluster)
		}
	}
	return nil
}

// uninstallCluster is supposed to do all the legwork in order to start from
// a functioning nomos cluster, and end with a cluster with all nomos-related
// additions removed.
func (i *Installer) uninstallCluster() error {
	kc := kubectl.New(context.Background())
	// Remove namespace
	if err := kc.DeleteNamespace(defaultNamespace); err != nil {
		return errors.Wrapf(err, "while removing namespace %q", defaultNamespace)
	}
	if err := kc.WaitForNamespaceDeleted(defaultNamespace); err != nil {
		return errors.Wrapf(err, "while waiting for namespace %q to disappear", defaultNamespace)
	}
	// Remove all managed cluster role bindings.  The code below assumes that
	// roles and role bindings are named the same, which is currently true.
	for _, name := range deploymentClusterRolesAndBindings {
		if err := kc.DeleteClusterrolebinding(name); err != nil {
			return errors.Wrapf(err, "while uninstalling cluster")
		}
		if err := kc.DeleteClusterrole(name); err != nil {
			return errors.Wrapf(err, "while uninstalling cluster")
		}
	}
	// TODO(filmil): Remove any other cluster-level nomos resources.
	return nil
}

// Uninstall uninstalls the system from the cluster.  Uninstall is asynchronous,
// so the uninstalled system will remain for a while after this completes.
func (i *Installer) Uninstall(yesIAmSure bool) error {
	if !yesIAmSure {
		return errors.Errorf("Please supply the flag --yes to proceed.")
	}
	if len(i.c.Contexts) == 0 {
		return errors.Errorf("no clusters requested")
	}
	cl, err := kubectl.LocalClusters()
	defer func() {
		if err := restoreContext(cl.Current); err != nil {
			glog.Errorf("while restoring context: %q: %v", cl.Current, err)
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "while getting local list of clusters")
	}
	kc := kubectl.New(context.Background())
	for _, cluster := range i.c.Contexts {
		glog.Infof("processing cluster: %q", cluster)
		err := kc.SetContext(cluster)
		if err != nil {
			return errors.Wrapf(err, "while setting context: %q", cluster)
		}
		err = i.uninstallCluster()
		if err != nil {
			return errors.Wrapf(err, "while processing cluster: %q", cluster)
		}
	}
	return nil
}
