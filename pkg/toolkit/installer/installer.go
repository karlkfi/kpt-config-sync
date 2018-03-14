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

	"github.com/golang/glog"
	"github.com/google/stolos/pkg/toolkit/bash"
	"github.com/google/stolos/pkg/toolkit/installer/config"
	"github.com/google/stolos/pkg/toolkit/kubectl"
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
	certsDirectory = "generated_certs"

	// certScript is a script for certificate generation.  Should be replaced
	// by go-native generation script.
	certScript = "generate-resourcequota-admission-controller-certs.sh"

	// admissionControllerScript is a admissionControllerScript used to deploy
	// admission controller specific secrets.
	admissionControllerScript = "deploy-resourcequota-admission-controller.sh"

	// defaultNamespace is the namespace to place resources into.
	defaultNamespace = "stolos-system"
)

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
	kc.Apply(applyDir)
	return nil
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
	c.CreateSecretGenericFromFile(
		secret, defaultNamespace,
		fmt.Sprintf("ssh=%v", i.c.Ssh.PrivateKeyFilename),
		fmt.Sprintf("known_hosts=%v", i.c.Ssh.KnownHostsFilename))
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
	if err := c.DeleteSecret(configmap, defaultNamespace); err != nil {
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

// processCluster installs the necessary files on the currently active cluster.
func (i *Installer) processCluster() error {
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
	// The common manifests need to be applied first, as they create the
	// namespace.
	err = i.applyAll(filepath.Join(i.workDir, "manifests/common"))
	if err != nil {
		return errors.Wrapf(err, "while applying manifests/common")
	}
	err = i.deployGitConfig()
	if err != nil {
		return errors.Wrapf(err, "processCluster")
	}
	err = i.deploySecrets()
	if err != nil {
		return errors.Wrapf(err, "processCluster")
	}
	err = i.applyAll(filepath.Join(i.workDir, "manifests/enrolled"))
	if err != nil {
		return errors.Wrapf(err, "while applying manifests/enrolled")
	}
	err = i.applyAll(filepath.Join(i.workDir, "yaml"))
	if err != nil {
		return errors.Wrapf(err, "while applying yaml")
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
	if len(i.c.Clusters) == 0 {
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
	i.createCertificates()
	kc := kubectl.New(context.Background())
	for _, cluster := range i.c.Clusters {
		glog.Infof("processing cluster: %q", cluster)
		err := kc.SetContext(cluster)
		if err != nil {
			return errors.Wrapf(err, "while setting context: %q", cluster)
		}
		// The processed cluster is set through the context use.
		err = i.processCluster()
		if err != nil {
			return errors.Wrapf(err, "while processing cluster: %q", cluster)
		}
	}
	return nil
}
