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
	"github.com/google/nomos/pkg/process/kubectl"
	"github.com/pkg/errors"
)

const (
	// defaultNamespace is the namespace to place resources into.
	defaultNamespace = "nomos-system"

	// deploymentTimeout is the default timeout to wait for each Nomos
	// component to become functional.  Components initialize in parallel, so
	// we expect that it's unlikely a wait would be
	// deploymentTimeout*len(deploymentComponents).
	deploymentTimeout = 3 * time.Minute

	// "Open sesame" for uninstallation.
	confirmUninstall = "deletedeletedelete"
)

var (
	// deploymentComponents are the components that are expected to be running
	// after installer completes.
	deploymentComponents = []string{
		fmt.Sprintf("%v:git-policy-importer", defaultNamespace),
		fmt.Sprintf("%v:resourcequota-admission-controller", defaultNamespace),
		fmt.Sprintf("%v:policy-admission-controller", defaultNamespace),
		fmt.Sprintf("%v:syncer", defaultNamespace),
	}

	// deploymentClusterRolesAndBindings are the cluster roles and
	// clusterrolebindings names created for the nomos system.
	deploymentClusterRolesAndBindings = []string{
		"nomos.dev:policy-importer",
		"nomos.dev:resourcequota-admission-controller",
		"nomos.dev:policy-admission-controller",
		"nomos.dev:syncer",

		// TODO(2018-05-31): these are legacy names, remove these at 0.9.0 <= release.
		"nomos-policy-importer",
		"nomos-resourcequota-admission-controller",
		"nomos-policy-admission-controller",
		"nomos-syncer",
	}

	// validatingWebhookConfigurations is a list of cluster-level validating
	// webhook configurations created by our admission controllers.
	validatingWebhookConfigurations = []string{
		"resource-quota.nomos.dev",
		"policy.nomos.dev",
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

	// The installers for creating certificates and secrets for admission
	// controllers.
	certInstallers []*certInstaller
}

// New returns a new Installer instance.
func New(c config.Config, workDir string) *Installer {
	resourceQuotaCertsInstaller := &certInstaller{
		generateScript: "generate-resourcequota-admission-controller-certs.sh",
		deployScript:   "deploy-resourcequota-admission-controller.sh",
		subDir:         "resourcequota",
	}
	policyNodesCertsInstaller := &certInstaller{
		generateScript: "generate-policy-admission-controller-certs.sh",
		deployScript:   "deploy-policy-admission-controller.sh",
		subDir:         "policynodes",
	}
	return &Installer{
		c:       c,
		workDir: workDir,
		certInstallers: []*certInstaller{
			resourceQuotaCertsInstaller,
			policyNodesCertsInstaller,
		},
	}
}

// createCertificates creates the certificates needed to bootstrap the syncing
// process.
func (i *Installer) createCertificates() error {
	for _, certInstaller := range i.certInstallers {
		glog.V(5).Infof("createCertificates: creating %s certificates", certInstaller.name())
		if err := certInstaller.createCertificates(i.workDir); err != nil {
			return err
		}
	}

	return nil
}

// applyAll runs 'kubectl apply -f applyDir'.
func (i *Installer) applyAll(applyDir string) error {
	kc := kubectl.New(context.Background())
	glog.V(5).Infof("Applying YAML files from directory: %v", applyDir)
	fi, err := os.Stat(applyDir)
	if err != nil {
		return errors.Wrapf(err, "applyAll: stat %v", applyDir)
	}
	if !fi.IsDir() {
		return errors.Errorf("applyAll: not a directory: %v", applyDir)
	}
	return kc.Apply(applyDir)
}

func (i *Installer) deployGitSecrets() error {
	const secret = "git-creds"
	glog.V(5).Info("deployGitSecrets: enter")
	var filenames []string
	if i.c.Git.UseSSH {
		filenames = append(filenames,
			fmt.Sprintf("ssh=%v", i.c.Git.PrivateKeyFilename))
		if i.c.Git.KnownHostsFilename != "" {
			filenames = append(filenames,
				fmt.Sprintf("known_hosts=%v", i.c.Git.KnownHostsFilename))
		}
	} else if i.c.Git.CookieFilename != "" {
		filenames = append(filenames,
			fmt.Sprintf("cookie_file=%v", i.c.Git.CookieFilename))
	} else {
		glog.V(5).Info("no PrivateKeyFilename, deploying empty secret")
	}
	c := kubectl.New(context.Background())
	if err := c.DeleteSecret(secret, defaultNamespace); err != nil {
		glog.V(5).Infof("failed to delete secret: %v", err)
	}

	if err := c.CreateSecretGenericFromFile(
		secret, defaultNamespace, filenames...); err != nil {
		return errors.Wrapf(err, "while creating ssh secrets")
	}
	return nil
}

func (i *Installer) deployGitConfig() error {
	const configmap = "git-policy-importer"
	glog.V(10).Info("deployGitConfig: enter")
	if i.c.Git.Empty() {
		glog.V(5).Info("Not deploying git configuration, no config specified.")
		return nil
	}
	c := kubectl.New(context.Background())
	if err := c.DeleteConfigmap(configmap, defaultNamespace); err != nil {
		glog.V(5).Infof("Failed to delete configmap: %v", err)
	}

	if err := c.CreateConfigmapFromLiterals(
		configmap, defaultNamespace, i.getGitConfigMapData()...,
	); err != nil {
		return errors.Wrapf(err, "while creating configmap git-policy-importer")
	}
	return nil
}

func (i *Installer) getGitConfigMapData() []string {
	return []string{
		fmt.Sprintf("GIT_SYNC_SSH=%v", i.c.Git.UseSSH),
		fmt.Sprintf("GIT_SYNC_REPO=%v", i.c.Git.SyncRepo),
		fmt.Sprintf("GIT_SYNC_BRANCH=%v", i.c.Git.SyncBranch),
		fmt.Sprintf("GIT_SYNC_WAIT=%v", i.c.Git.SyncWaitSeconds),
		fmt.Sprintf("GIT_KNOWN_HOSTS=%v", i.c.Git.KnownHostsFilename != ""),
		fmt.Sprintf("GIT_COOKIE_FILE=%v", i.c.Git.CookieFilename != ""),
		fmt.Sprintf("POLICY_DIR=%v", i.c.Git.RootPolicyDir),
	}
}

func (i *Installer) deploySecrets() error {
	glog.V(5).Info("deploySecrets: enter")
	err := i.deployGitSecrets()
	if err != nil {
		return errors.Wrapf(err, "while deploying git secrets")
	}
	for _, certInstaller := range i.certInstallers {
		if err := certInstaller.deploySecrets(i.workDir); err != nil {
			return errors.Wrapf(err, "while deploying %s secrets", certInstaller.name())
		}
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
			if err = i.removeClusterAdmin(i.c.User); err != nil {
				glog.Warningf("could not remove cluster admin role for user: %v: %v", i.c.User, err)
			}
		}()
	}
	c := kubectl.New(context.Background())

	if err = i.checkVersion(c); err != nil {
		return errors.Wrapf(err, "while checking version for context")
	}
	if err = i.c.Validate(config.OsFileExists{}); err != nil {
		return errors.Wrapf(err, "while validating the configuration")
	}
	// Delete the git policy importer deployment.  This is important because a
	// change in the git creds should also be reflected in the importer.
	if err = c.DeleteDeployment("git-policy-importer", defaultNamespace); err != nil {
		return errors.Wrapf(err, "while deleting git-policy-importer deployment")
	}
	if err = i.applyAll(filepath.Join(i.workDir, "manifests")); err != nil {
		return errors.Wrapf(err, "while applying manifests")
	}
	if err = i.deployGitConfig(); err != nil {
		return errors.Wrapf(err, "processCluster")
	}
	if err = i.deploySecrets(); err != nil {
		return errors.Wrapf(err, "processCluster")
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
// if useCurrent is set, and the list of clusters to install is empty, it will
// use the current context to install.
func (i *Installer) Run(useCurrent bool) error {
	cl, err := kubectl.LocalClusters()
	defer func() {
		if err := restoreContext(cl.Current); err != nil {
			glog.Errorf("while restoring context: %q: %v", cl.Current, err)
		}
	}()
	if len(i.c.Contexts) == 0 {
		if useCurrent && cl.Current != "" {
			i.c.Contexts = []string{cl.Current}
		} else {
			return errors.Errorf("no clusters requested for installation")
		}
	}

	if err != nil {
		return errors.Wrapf(err, "while getting local list of clusters")
	}
	if err := i.createCertificates(); err != nil {
		return errors.Wrapf(err, "while creating certificates")
	}
	kc := kubectl.New(context.Background())
	for _, cluster := range i.c.Contexts {
		glog.Infof("Setting up nomos on cluster: %q", cluster)
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
	// Remove admission webhooks.
	for _, w := range validatingWebhookConfigurations {
		if err := kc.DeleteValidatingWebhookConfiguration(w); err != nil {
			return errors.Wrapf(err, "while deleting webhook configurations")
		}
	}
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
		// Ignore errors while deleting, but log them.
		if err := kc.DeleteClusterrolebinding(name); err != nil {
			glog.Errorf("%v", errors.Wrapf(err, "while uninstalling cluster"))
		}
		if err := kc.DeleteClusterrole(name); err != nil {
			glog.Errorf("%v", errors.Wrapf(err, "while uninstalling cluster"))
		}
	}
	// TODO(filmil): Remove any other cluster-level nomos resources.
	return nil
}

// Uninstall uninstalls the system from the cluster.  Uninstall is asynchronous,
// so the uninstalled system will remain for a while after this completes.
func (i *Installer) Uninstall(confirm string) error {
	if confirm != confirmUninstall {
		return errors.Errorf("to confirm uninstall (destructive) set -uninstall=%q", confirmUninstall)
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
