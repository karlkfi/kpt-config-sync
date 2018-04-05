// Package config contains the shared configuration for the installer script.
package config

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

const (
	defaultSyncWaitTimeoutSeconds = 15
	defaultSyncBranch             = "master"
)

// DefaultConfig contains the empty default values for a Config.
var DefaultConfig = Config{
	Git: GitConfig{
		SyncWaitSeconds: defaultSyncWaitTimeoutSeconds,
		SyncBranch:      defaultSyncBranch,
	},
}

// CertConfig contains the configuration that pertains to the certificate
// authority to be used for the cluster.  If defined,
type CertConfig struct {
	// CaCertFilename is the name of the file containing the PEM-encoded
	// CA certificate
	CaCertFilename string `json:"caCertFilename,omitempty"`

	// CaKeyFilename is the name of the file containing the PEM-encoded
	// private key of the certificate authority.  This will be used to
	// sign the generated certificate but otherwise won't be stored.  Please
	// make sure that this file has correct permissions.
	CaKeyFilename string `json:"caKeyFilename,omitempty"`
}

// SshConfig contains the private key filename and the known hosts filename
// to add to the cluster.  Omitted if unused.
type SshConfig struct {
	// PrivateKeyFilename is the filename containing the private key used
	// for SSH authentication.  This entry is communicated through a file
	// to avoid exposing the contents of the secret in the process table.
	// If this entry is empty, SSH will be set to false.
	PrivateKeyFilename string `json:"privateKeyFilename,omitempty"`

	// KnownHostsFilename is the filename containing the known hosts SSH
	// file that Kubernetes will use.  Not copied if not defined.
	KnownHostsFilename string `json:"knownHostsFilename,omitempty"`
}

// GitConfig contains the settings for the git importer repository.
type GitConfig struct {
	// SyncRepo is the git repository name to sync from.
	SyncRepo string `json:"GIT_SYNC_REPO"`

	// SyncBranch is the branch to sync from.  Default: "master".
	SyncBranch string `json:"GIT_SYNC_BRANCH"`

	// RootPolicyDir is the absolute path of the directory that contains
	// the local policy.  Default: the root directory of the repo.
	RootPolicyDir string `json:"ROOT_POLICY_DIR"`

	// SyncWaitSeconds is the time duration in seconds between consecutive
	// syncs.  Default: 15 seconds.
	SyncWaitSeconds int64 `json:"GIT_SYNC_WAIT"`
}

// Config contains the configuration for the installer.  The install process
// is made based on this configuration.
type Config struct {
	// The user account that will drive the installation.  Required to insert
	// cluster administration role bindings into GKE clusters.
	User string `json:"user,omitempty"`

	// Contexts contains the names of the cluster contexts to attempt to install
	// into.
	Contexts []string `json:"contexts,omitempty"`

	// Git contains the git-specific configuration.
	Git GitConfig `json:"git,omitempty"`

	// CertConfig contains the CA information for the CA to be used to generate
	// cluster configuration.  If left undefined, the install process will
	// generate a self-signed CA certificate to use.
	CertConfig CertConfig `json:"certConfig,omitempty"`

	// Ssh contains the SSH configuration to use.  If omitted, assumes that
	// SSH is not used.
	Ssh SshConfig `json:"ssh,omitempty"`
}

// Load loads configuration from a reader in either YAML or JSON format.
func Load(r io.Reader) (Config, error) {
	var c Config
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return Config{}, errors.Wrapf(err, "while reading config")
	}
	c = DefaultConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return Config{}, errors.Wrapf(err, "while loading configuration")
	}
	c.Ssh.KnownHostsFilename = strings.Replace(c.Ssh.KnownHostsFilename, "$HOME", "/home/user", 1)
	c.Ssh.PrivateKeyFilename = strings.Replace(c.Ssh.PrivateKeyFilename, "$HOME", "/home/user", 1)
	return c, nil
}

// WriteInto writes the configuration into supplied writer in JSON format.
func (c Config) WriteInto(w io.Writer) error {
	c.Ssh.KnownHostsFilename = strings.Replace(c.Ssh.KnownHostsFilename, "/home/user", "$HOME", 1)
	c.Ssh.PrivateKeyFilename = strings.Replace(c.Ssh.PrivateKeyFilename, "/home/user", "$HOME", 1)
	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrapf(err, "while marshalling config")
	}
	i, err := w.Write(b)
	if i != len(b) {
		return errors.Errorf("short write")
	}
	return err
}
