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
	"path/filepath"
	"time"

	"github.com/blang/semver"
	"github.com/golang/glog"
	"github.com/google/nomos/pkg/installer/config"
	"github.com/google/nomos/pkg/process/kubectl"
	"github.com/pkg/errors"

	"k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// defaultNamespace is the namespace to place resources into.
	defaultNamespace = "nomos-system"

	// deploymentTimeout is the default timeout to wait for each Nomos
	// component to become functional.  Components initialize in parallel, so
	// we expect that it's unlikely a wait would be
	// deploymentTimeout*len(deployments).
	deploymentTimeout = 3 * time.Minute

	// nomosLabel is a label applied to nomos-created resources not under default namespace
	nomosLabel = "nomos.dev/system"

	// "Open sesame" for uninstallation.
	confirmUninstall = "deletedeletedelete"

	syncerConfigMapName = "syncer"

	gitPolicyImporterDeployment = "git-policy-importer"
	gcpPolicyImporterDeployment = "gcp-policy-importer"

	gitPolicyImporterConfigMap = "git-policy-importer"
	gcpPolicyImporterConfigMap = "gcp-policy-importer"

	gitPolicyImporterCreds = "git-creds"
	gcpPolicyImporterCreds = "gcp-creds"
)

var (
	// commonDeployments are the common deployments managed by installer.
	commonDeployments = []string{
		"monitor",
		"policy-admission-controller",
		"syncer",
	}

	// gitDeployments is the deployments only used on Git-configured deployments
	gitDeployments = []string{
		"resourcequota-admission-controller",
	}

	// mv is the minimum supported cluster version.  It is not possible to install
	// on an earlier cluster due to missing features.
	mv = semver.MustParse("1.9.0")
)

// Installer is the process that runs the system installation.
type Installer struct {
	// The configuration that the installer works out of.
	c config.Config

	// Kubernetes context.
	k *kubectl.Context

	// The working directory of the installer.
	workDir string

	// The installers for creating certificates and secrets for admission
	// controllers.
	certInstallers []*certInstaller
}

// New returns a new Installer instance.
func New(c config.Config, workDir string) *Installer {
	policyNodesCertsInstaller := &certInstaller{
		generateScript: "generate-policy-admission-controller-certs.sh",
		deployScript:   "deploy-policy-admission-controller.sh",
		subDir:         "policynodes",
	}
	ins := &Installer{
		c:       c,
		k:       kubectl.New(context.Background()),
		workDir: workDir,
		certInstallers: []*certInstaller{
			policyNodesCertsInstaller,
		},
	}
	// we don't use resource quota control in GCP installs
	if !c.Git.Empty() {
		resourceQuotaCertsInstaller := &certInstaller{
			generateScript: "generate-resourcequota-admission-controller-certs.sh",
			deployScript:   "deploy-resourcequota-admission-controller.sh",
			subDir:         "resourcequota",
		}
		ins.certInstallers = append(ins.certInstallers, resourceQuotaCertsInstaller)
	}
	return ins
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

func (i *Installer) deployConfigMap(name string, content []string) error {
	if err := i.k.DeleteConfigMap(name, defaultNamespace); err != nil {
		glog.V(5).Infof("Failed to delete configmap: %v", err)
	}

	if err := i.k.CreateConfigmapFromLiterals(name, defaultNamespace, content...); err != nil {
		return errors.Wrapf(err, "while creating configmap %s", name)
	}
	return nil
}

func (i *Installer) gitConfigMapContent() []string {
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

func (i *Installer) gcpConfigMapContent() []string {
	c := []string{
		fmt.Sprintf("ORG_ID=%v", i.c.GCP.OrgID),
	}
	if i.c.GCP.PolicyAPIAddress != "" {
		c = append(c, fmt.Sprintf("POLICY_API_ADDRESS=%v", i.c.GCP.PolicyAPIAddress))
	}
	return c
}

// syncerConfigMapContent returns a list of listerals defining the ConfigMap for the syncer
func (i *Installer) syncerConfigMapContent() []string {
	return []string{
		fmt.Sprintf("gcp.mode=%v", !i.c.GCP.Empty()),
	}
}

func (i *Installer) deploySecret(name string, content []string) error {
	if err := i.k.DeleteSecret(name, defaultNamespace); err != nil {
		glog.V(5).Infof("failed to delete secret %s: %v", name, err)
	}
	if err := i.k.CreateSecretGenericFromFile(
		name, defaultNamespace, content...); err != nil {
		return errors.Wrapf(err, "failed to create secret %s", name)
	}
	return nil
}

// gitSecretContent returns the content of the Secret consumed by GitPolicyImporter.
func (i *Installer) gitSecretContent() []string {
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
	return filenames
}

// gitSecretContent returns the content of the Secret consumed by GCPPolicyImporter.
func (i *Installer) gcpSecretContent() []string {
	var filenames []string
	filenames = append(filenames,
		fmt.Sprintf("gcp-private-key=%v", i.c.GCP.PrivateKeyFilename))
	return filenames
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

// checkContexts examines whether contexts have been provided, and, if they have not and useCurrent
// is true and cl.Current provided, uses the current context.
func (i *Installer) checkContexts(cl kubectl.ClusterList, useCurrent bool) error {
	if len(i.c.Contexts) == 0 {
		if useCurrent && cl.Current != "" {
			i.c.Contexts = []string{cl.Current}
		} else {
			return errors.Errorf("no clusters requested")
		}
	}
	return nil
}

// processCluster installs the necessary files on the currently active cluster.
// In addition the current cluster context is passed in.
func (i *Installer) processCluster(cluster string) error {
	var err error
	glog.V(5).Info("processCluster: enter")

	if i.c.User != "" {
		if err = i.k.AddClusterAdmin(i.c.User); err != nil {
			return errors.Wrapf(err, "could not make %v the cluster admin.", i.c.User)
		}
		defer func() {
			// Ensure that this is ran at end of cluster process, irrespective
			// of whether the install was successful.
			if err = i.k.RemoveClusterAdmin(i.c.User); err != nil {
				glog.Warningf("could not remove cluster admin role for user: %v: %v", i.c.User, err)
			}
		}()
	}

	if err = i.checkVersion(i.k); err != nil {
		return errors.Wrapf(err, "while checking version for context")
	}
	if err = i.c.Validate(config.OsFileExists{}); err != nil {
		return errors.Wrapf(err, "while validating the configuration")
	}
	if err = i.k.Apply(filepath.Join(i.workDir, "manifests")); err != nil {
		return errors.Wrapf(err, "while applying manifests")
	}

	var importerDeployment, importerConfigMapName, importerSecretName string
	var importerConfigMapContent, importerSecretContent []string
	if !i.c.Git.Empty() {
		importerDeployment = gitPolicyImporterDeployment
		importerConfigMapName = gitPolicyImporterConfigMap
		importerConfigMapContent = i.gitConfigMapContent()
		importerSecretName = gitPolicyImporterCreds
		importerSecretContent = i.gitSecretContent()
	} else {
		importerDeployment = gcpPolicyImporterDeployment
		importerConfigMapName = gcpPolicyImporterConfigMap
		importerConfigMapContent = i.gcpConfigMapContent()
		importerSecretName = gcpPolicyImporterCreds
		importerSecretContent = i.gcpSecretContent()
	}
	syncerConfigMapContent := i.syncerConfigMapContent()

	// Delete the importer deployment.  This is important because a
	// change in the secret should also be reflected in the importer.
	if err = i.k.DeleteDeployment(importerDeployment, defaultNamespace); err != nil {
		return errors.Wrapf(err, "while deleting Deployment %s", importerDeployment)
	}
	if err = i.deployConfigMap(importerConfigMapName, importerConfigMapContent); err != nil {
		return errors.Wrapf(err, "failed to create importer ConfigMap: %v", importerConfigMapName)
	}
	if err = i.deploySecret(importerSecretName, importerSecretContent); err != nil {
		return errors.Wrapf(err, "failed to create Secret: %v", importerSecretName)
	}
	if err = i.deployConfigMap(syncerConfigMapName, syncerConfigMapContent); err != nil {
		return errors.Wrapf(err, "failed to create syncer ConfigMap: %v", syncerConfigMapContent)
	}
	for _, certInstaller := range i.certInstallers {
		if err = certInstaller.deploySecrets(i.workDir); err != nil {
			return errors.Wrapf(err, "while deploying %s Secrets", certInstaller.name())
		}
	}
	var deployments []string
	deployments = append(deployments, commonDeployments...)
	deployments = append(deployments, importerDeployment)
	if !i.c.Git.Empty() {
		deployments = append(deployments, gitDeployments...)
	}
	for _, d := range deployments {
		if err = i.k.Apply(filepath.Join(i.workDir, "manifests/deployment", fmt.Sprintf("%s.yaml", d))); err != nil {
			return errors.Wrapf(err, "while applying Deployment: %s", d)
		}
	}
	if err = i.k.WaitForDeployments(deploymentTimeout, defaultNamespace, deployments...); err != nil {
		return errors.Wrapf(err, "while waiting for system components")
	}
	return err
}

// Run starts the installer process, and reports error at the process end, if any.
// if useCurrent is set, and the list of clusters to install is empty, it will
// use the current context to install.
func (i *Installer) Run(useCurrent bool) error {
	cl, err := kubectl.LocalClusters()
	defer func() {
		if err2 := i.k.SetContext(cl.Current); err2 != nil {
			glog.Errorf("while restoring context: %q: %v", cl.Current, err2)
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "while getting local list of clusters")
	}
	err = i.checkContexts(cl, useCurrent)
	if err != nil {
		return errors.Wrapf(err, "while checking cluster context")
	}
	if err := i.createCertificates(); err != nil {
		return errors.Wrapf(err, "while creating certificates")
	}
	for _, cluster := range i.c.Contexts {
		glog.Infof("Setting up nomos on cluster: %q", cluster)
		err := i.k.SetContext(cluster)
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

// DeletePolicyNodes deletes all policynodes that are in the hierarchy. If an error is encountered
// while listing or deleting a node, the DeletePolicyNodes terminates immediately.
func (i *Installer) DeletePolicyNodes() error {
	pn, err := i.k.PolicyHierarchy().NomosV1().PolicyNodes().List(metav1.ListOptions{
		IncludeUninitialized: true,
	})

	if err != nil {
		return errors.Wrapf(err, "error listing policynodes")
	}

	for _, n := range pn.Items {
		err = i.k.PolicyHierarchy().NomosV1().PolicyNodes().Delete(n.Name, metav1.NewDeleteOptions(0))
		if err != nil {
			return errors.Wrapf(err, "error deleting policynode: %s", n.Name)
		}
	}
	return nil
}

// DeleteClusterPolicies deletes all cluster policies in the Hierarchy. If an error is encountered
// while listing or deleting a node, the DeletePolicyNodes terminates immediately.
func (i *Installer) DeleteClusterPolicies() error {
	cp, err := i.k.PolicyHierarchy().NomosV1().ClusterPolicies().List(metav1.ListOptions{
		IncludeUninitialized: true,
	})

	if err != nil {
		return errors.Wrapf(err, "error listing clusterpolicies")
	}

	for _, p := range cp.Items {
		err = i.k.PolicyHierarchy().NomosV1().ClusterPolicies().Delete(p.Name, metav1.NewDeleteOptions(0))
		if err != nil {
			return errors.Wrapf(err, "error deleting clusterpolicy: %s", p.Name)
		}
	}
	return nil
}

// names returns an array of ClusterRoleBinding and ClusterRole names that are in the input
// lists. The function does not output duplicates, and does not do any validation of the similarity
// of the two input lists.
func names(crbl *v1.ClusterRoleBindingList, crl *v1.ClusterRoleList) []string {
	nameMap := map[string]bool{}
	for _, v := range crbl.Items {
		if !nameMap[v.Name] {
			nameMap[v.Name] = true
		}
	}
	for _, v := range crl.Items {
		if !nameMap[v.Name] {
			nameMap[v.Name] = true
		}
	}
	var names []string
	for k := range nameMap {
		names = append(names, k)
	}
	return names
}

// uninstallCluster is supposed to do all the legwork in order to start from
// a functioning nomos cluster, and end with a cluster with all nomos-related
// additions removed.
func (i *Installer) uninstallCluster() error {
	// Remove admission webhooks.
	vwcToDelete, err := i.k.Kubernetes().AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().List(
		metav1.ListOptions{LabelSelector: nomosLabel})
	if err != nil {
		return errors.Wrapf(err, "while listing webhook configurations to delete")
	}
	for _, w := range vwcToDelete.Items {
		if err = i.k.DeleteValidatingWebhookConfiguration(w.Name); err != nil {
			return errors.Wrapf(err, "while deleting webhook configurations")
		}
	}
	// Remove namespace
	if err = i.k.DeleteNamespace(defaultNamespace); err != nil {
		return errors.Wrapf(err, "while removing namespace %q", defaultNamespace)
	}
	if err = i.k.WaitForNamespaceDeleted(defaultNamespace); err != nil {
		return errors.Wrapf(err, "while waiting for namespace %q to disappear", defaultNamespace)
	}
	// Remove all managed cluster role bindings.  The code below assumes that
	// roles and role bindings are named the same, which is currently true.
	crbToDelete, err := i.k.Kubernetes().RbacV1().ClusterRoleBindings().List(
		metav1.ListOptions{LabelSelector: nomosLabel})
	if err != nil {
		return errors.Wrapf(err, "while listing cluster role bindings to delete")
	}
	crToDelete, err := i.k.Kubernetes().RbacV1().ClusterRoles().List(metav1.ListOptions{LabelSelector: nomosLabel})
	if err != nil {
		return errors.Wrapf(err, "while listing cluster roles to delete")
	}
	dcrb := names(crbToDelete, crToDelete)
	for _, name := range dcrb {
		// Ignore errors while deleting, but log them.
		if err = i.k.DeleteClusterrolebinding(name); err != nil {
			glog.Errorf("%v", errors.Wrapf(err, "while deleting cluster role binding"))
		}
		if err = i.k.DeleteClusterrole(name); err != nil {
			glog.Errorf("%v", errors.Wrapf(err, "while deleting cluster role"))
		}
	}
	if err = i.DeletePolicyNodes(); err != nil {
		return err
	}
	if err = i.DeleteClusterPolicies(); err != nil {
		return err
	}
	return nil
}

// Uninstall uninstalls the system from the cluster.  Uninstall is asynchronous,
// so the uninstalled system will remain for a while after this completes.
// The correct confirm string must be provided for uninstallation to proceed.
// If useCurrent is set, and the list of clusters to install is empty, Uninstall will
// uninstall the cluster in the current context.
func (i *Installer) Uninstall(confirm string, useCurrent bool) error {
	if confirm != confirmUninstall {
		return errors.Errorf("to confirm uninstall (destructive) set -uninstall=%q", confirmUninstall)
	}
	cl, err := kubectl.LocalClusters()
	defer func() {
		if err2 := i.k.SetContext(cl.Current); err2 != nil {
			glog.Errorf("while restoring context: %q: %v", cl.Current, err2)
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "while getting local list of clusters")
	}
	err = i.checkContexts(cl, useCurrent)
	if err != nil {
		return errors.Wrapf(err, "while checking cluster context")
	}
	for _, cluster := range i.c.Contexts {
		glog.Infof("processing cluster: %q", cluster)
		err := i.k.SetContext(cluster)
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
