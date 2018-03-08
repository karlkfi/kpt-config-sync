// Package config contains the shared configuration for the installer script.
package config

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const (
	defaultSyncWaitTimeoutSeconds = 15
	defaultSyncBranch             = "master"
)

var defaultConfig = Config{
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
	CaCertFilename string `json:"caCertFilename"`

	// CaKeyFilename is the name of the file containing the PEM-encoded
	// private key of the certificate authority.  This will be used to
	// sign the generated certificate but otherwise won't be stored.  Please
	// make sure that this file has correct permissions.
	CaKeyFilename string `json:"caKeyFilename"`
}

// SshConfig contains the private key filename and the known hosts filename
// to add to the cluster.  Omitted if unused.
type SshConfig struct {
	// PrivateKeyFilename is the filename containing the private key used
	// for SSH authentication.  This entry is communicated through a file
	// to avoid exposing the contents of the secret in the process table.
	PrivateKeyFilename string `json:"privateKeyFilename"`

	// KnownHostsFilename is the filename containing the known hosts SSH
	// file that Kubernetes will use.  Not copied if not defined.
	KnownHostsFilename string `json:"knownHostsFilename,omitempty"`
}

// GitConfig contains the settings for the git importer repository.
type GitConfig struct {
	// SyncRepo is the git repository name to sync from.
	SyncRepo string `json:"syncRepo"`

	// SyncBranch is the branch to sync from.  Default: "master".
	SyncBranch string `json:"syncBranch"`

	// RootPolicyDir is the absolute path of the directory that contains
	// the local policy.  Default: the root directory of the repo.
	RootPolicyDir string `json:"rootPolicyDir"`

	// SyncWaitSeconds is the time duration in seconds between consecutive
	// syncs.  Default: 15 seconds.
	SyncWaitSeconds int64 `json:"syncWaitSeconds"`
}

// Config contains the configuration for the installer.  The install process
// is made based on this configuration.
type Config struct {
	// Clusters contains the names of the cluster contexts to attempt to install
	// into.
	Clusters []string `json:"clusters,omitempty"`

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

// Load loads configuration from a reader in JSON format, or returns an error
// in case of failure.
func Load(r io.Reader) (Config, error) {
	var c Config
	c = defaultConfig
	d := json.NewDecoder(r)
	if err := d.Decode(&c); err != nil {
		return Config{}, errors.Wrapf(err, "while loading configuration")
	}
	c.Ssh.KnownHostsFilename = strings.Replace(c.Ssh.KnownHostsFilename, "$HOME", "/home/user", 1)
	c.Ssh.PrivateKeyFilename = strings.Replace(c.Ssh.PrivateKeyFilename, "$HOME", "/home/user", 1)
	return c, nil
}

// WriteInto writes the configuration into supplied writer in JSON format.
func (c Config) WriteInto(w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "    ")
	c.Ssh.KnownHostsFilename = strings.Replace(c.Ssh.KnownHostsFilename, "/home/user", "$HOME", 1)
	c.Ssh.PrivateKeyFilename = strings.Replace(c.Ssh.PrivateKeyFilename, "/home/user", "$HOME", 1)
	err := e.Encode(c)
	if err != nil {
		return errors.Wrapf(err, "while writing config")
	}
	return nil
}
